package basic

import (
	"encoding/json"
	"errors"
	"io/fs"
	"mime"
	"path"
	"strings"

	docs "handfree-work/octo-backup/docs"

	"github.com/gofiber/fiber/v3"
	swaggerFiles "github.com/swaggo/files/v2"
)

const swaggerIndex = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>OctoBackup API 文档</title>
  <link rel="stylesheet" href="./swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="./swagger-ui-bundle.js"></script>
  <script src="./swagger-ui-standalone-preset.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "./doc.json",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
      layout: "StandaloneLayout"
    });
  </script>
</body>
</html>`

// Register 注册 basic 模块的 Swagger UI 与 OpenAPI JSON 文档。
func Register(app *fiber.App) {
	app.Get("/swagger", func(c fiber.Ctx) error {
		return c.Redirect().Status(fiber.StatusMovedPermanently).To("/swagger/index.html")
	})
	app.Get("/swagger/", func(c fiber.Ctx) error {
		return c.Redirect().Status(fiber.StatusMovedPermanently).To("/swagger/index.html")
	})
	app.Get("/swagger/doc.json", func(c fiber.Ctx) error {
		document, err := swaggerDocument()
		if err != nil {
			return err
		}
		return c.Type("json").SendString(document)
	})
	app.Get("/swagger/index.html", func(c fiber.Ctx) error {
		return c.Type("html").SendString(swaggerIndex)
	})
	app.Get("/swagger/*", swaggerAsset)
}

func swaggerDocument() (string, error) {
	document := docs.SwaggerInfo.ReadDoc()
	var specification map[string]any
	if err := json.Unmarshal([]byte(document), &specification); err != nil {
		return "", err
	}
	delete(specification, "host")
	encoded, err := json.Marshal(specification)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func swaggerAsset(c fiber.Ctx) error {
	fileName := path.Clean(strings.TrimPrefix(c.Params("*"), "/"))
	if fileName == "." || strings.HasPrefix(fileName, "../") {
		return c.SendStatus(fiber.StatusNotFound)
	}
	content, err := fs.ReadFile(swaggerFiles.FS, fileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return err
	}
	if contentType := mime.TypeByExtension(path.Ext(fileName)); contentType != "" {
		c.Set(fiber.HeaderContentType, contentType)
	}
	return c.Send(content)
}
