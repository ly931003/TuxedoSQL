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

	// 注册所有支持的数据库驱动及其对应的模式内省器
	// key 与 Connection.Driver 字段值匹配
	connManager := repository.NewConnectionManager(connRepo,
		map[string]repository.DatabaseDriver{
			"mysql":    &repository.MySQLDriver{},
			"postgres": &repository.PostgresDriver{},
			"sqlite":   &repository.SQLiteDriver{},
		},
		map[string]repository.SchemaIntrospector{
			"mysql":    &repository.MySQLSchema{},
			"postgres": &repository.PostgresSchema{},
			"sqlite":   &repository.SQLiteSchema{},
		},
	)
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
