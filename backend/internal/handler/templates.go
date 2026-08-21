package handler

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"xorapi/internal/service"
	"xorapi/internal/store"

	"github.com/gin-gonic/gin"
)

type TemplateManifest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Description string `json:"description"`
}

func templatesDir() string {
	dir := "./data/templates"
	if store.Cfg != nil && store.Cfg.Server.DataDir != "" {
		dir = filepath.Join(store.Cfg.Server.DataDir, "templates")
	}
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// TemplatesDir 模板根目录（供静态路由使用）
func TemplatesDir() string { return templatesDir() }

// AdminListTemplates 已安装模板列表
func AdminListTemplates(c *gin.Context) {
	dir := templatesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		OK(c, gin.H{"list": []any{}, "active": service.GetConfig("active_template")})
		return
	}
	list := []TemplateManifest{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), "manifest.json"))
		if err != nil {
			continue
		}
		var m TemplateManifest
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		if m.Slug == "" {
			m.Slug = e.Name()
		}
		list = append(list, m)
	}
	active := service.GetConfig("active_template")
	if active == "" {
		active = "default"
	}
	OK(c, gin.H{"list": list, "active": active})
}

var slugRe = regexp.MustCompile(`[^a-z0-9_-]+`)

// AdminUploadTemplate 上传模板 ZIP
func AdminUploadTemplate(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		Fail(c, http.StatusBadRequest, "请上传 ZIP 文件")
		return
	}
	defer file.Close()
	tmp, err := os.CreateTemp("", "xorapi-tpl-*.zip")
	if err != nil {
		Fail(c, http.StatusInternalServerError, "临时文件创建失败")
		return
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, file); err != nil {
		Fail(c, http.StatusBadRequest, "文件读取失败")
		return
	}
	tmp.Close()

	zr, err := zip.OpenReader(tmp.Name())
	if err != nil {
		Fail(c, http.StatusBadRequest, "不是合法的 ZIP 文件")
		return
	}
	defer zr.Close()

	dest, err := os.MkdirTemp("", "xorapi-tpl-dest-*")
	if err != nil {
		Fail(c, http.StatusInternalServerError, "临时目录创建失败")
		return
	}
	cleanup := func() { os.RemoveAll(dest) }

	var manifestPath string
	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		if name == "manifest.json" || strings.HasSuffix(name, "/manifest.json") {
			manifestPath = name
		}
		if err := extractZipFile(f, dest); err != nil {
			cleanup()
			Fail(c, http.StatusBadRequest, "解压失败: "+err.Error())
			return
		}
	}
	if manifestPath == "" {
		cleanup()
		Fail(c, http.StatusBadRequest, "ZIP 中缺少 manifest.json")
		return
	}
	baseDir := filepath.ToSlash(filepath.Dir(manifestPath))
	if baseDir == "." {
		baseDir = ""
	}
	root := filepath.Join(dest, filepath.FromSlash(baseDir))
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		cleanup()
		Fail(c, http.StatusBadRequest, "manifest.json 读取失败")
		return
	}
	var m TemplateManifest
	if err := json.Unmarshal(data, &m); err != nil || m.Name == "" {
		cleanup()
		Fail(c, http.StatusBadRequest, "manifest.json 格式错误（需包含 name 字段）")
		return
	}
	if _, err := os.Stat(filepath.Join(root, "index.html")); err != nil {
		cleanup()
		Fail(c, http.StatusBadRequest, "模板缺少入口 index.html")
		return
	}
	if m.Slug == "" {
		m.Slug = slugRe.ReplaceAllString(strings.ToLower(m.Name), "-")
	}
	if m.Slug == "" || m.Slug == "default" {
		m.Slug = "custom-" + fmt.Sprint(os.Getpid())
	}

	final := filepath.Join(templatesDir(), m.Slug)
	os.RemoveAll(final)
	if err := os.Rename(root, final); err != nil {
		cleanup()
		Fail(c, http.StatusInternalServerError, "模板安装失败: "+err.Error())
		return
	}
	cleanup()
	OK(c, gin.H{"slug": m.Slug, "name": m.Name})
}

// extractZipFile 带 zip-slip 防护的解压
func extractZipFile(f *zip.File, dest string) error {
	name := filepath.ToSlash(f.Name)
	if strings.Contains(name, "..") {
		return fmt.Errorf("非法路径: %s", name)
	}
	path := filepath.Join(dest, filepath.FromSlash(name))
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(dest)) {
		return fmt.Errorf("非法路径: %s", name)
	}
	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

// AdminActivateTemplate 激活模板
func AdminActivateTemplate(c *gin.Context) {
	var req struct {
		Slug string `json:"slug" binding:"required"`
	}
	if !BindJSON(c, &req) {
		return
	}
	if req.Slug != "default" {
		if _, err := os.Stat(filepath.Join(templatesDir(), req.Slug, "index.html")); err != nil {
			Fail(c, http.StatusBadRequest, "模板不存在或缺少 index.html")
			return
		}
	}
	if err := service.SetConfigs(store.DB, map[string]string{"active_template": req.Slug}); err != nil {
		Fail(c, http.StatusInternalServerError, "激活失败: "+err.Error())
		return
	}
	OK(c, gin.H{"active": req.Slug})
}
