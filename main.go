package main

import (
	"embed"
	"log"

	"tuxedosql/internal/repository"
	"tuxedosql/internal/service"
	"tuxedosql/pkg/fileutil"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	store, err := fileutil.NewJSONStore()
	if err != nil {
		log.Fatalf("初始化配置存储失败: %v", err)
	}

	connRepo := repository.NewConnectionRepository(store)
	tabRepo := repository.NewTabRepository(store)
	historyRepo := repository.NewHistoryRepository(store)
	connManager := repository.NewConnectionManager(connRepo)
	defer connManager.CloseAll()

	app := application.New(application.Options{
		Name:        "TuxedoSQL",
		Description: "数据库可视化管理工具",
		Services: []application.Service{
			application.NewService(service.NewConnectionService(connManager, connRepo)),
			application.NewService(service.NewQueryService(connManager, connRepo, tabRepo, historyRepo)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "TuxedoSQL",
		Width:            1200,
		Height:           800,
		MinWidth:         800,
		MinHeight:        500,
		BackgroundColour: application.NewRGB(255, 255, 255),
		URL:              "/",
	})

	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
