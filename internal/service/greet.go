package service

// GreetService 示例服务，演示如何暴露 Go 方法给前端调用。
// 在 main.go 中通过 application.NewService(&service.GreetService{}) 注册后，
// 前端即可通过生成的绑定 GreetService.Greet(name) 调用此方法。
type GreetService struct{}

// Greet 返回对给定名称的问候语。
func (g *GreetService) Greet(name string) string {
	return "Hello " + name + "!"
}
