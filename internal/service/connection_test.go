package service

import (
	"testing"

	"tuxedosql/internal/model"
	"tuxedosql/internal/repository"
	"tuxedosql/pkg/fileutil"
)

func newTestConnectionService() *ConnectionService {
	store, _ := fileutil.NewJSONStore()
	connRepo := repository.NewConnectionRepository(store)
	return NewConnectionService(nil, connRepo)
}

func TestConnectionService_Create(t *testing.T) {
	svc := newTestConnectionService()

	tests := []struct {
		name    string
		params  model.CreateConnectionParams
		wantErr bool
		checkFn func(t *testing.T, conn *model.Connection)
	}{
		{
			name: "创建有效连接",
			params: model.CreateConnectionParams{
				Name:     "测试连接",
				Host:     "127.0.0.1",
				Port:     3306,
				Username: "root",
				Password: "test",
				Database: "mysql",
				Timezone: "Asia/Shanghai",
			},
			wantErr: false,
			checkFn: func(t *testing.T, conn *model.Connection) {
				if conn.Timezone != "Asia/Shanghai" {
					t.Errorf("Timezone = %q, want %q", conn.Timezone, "Asia/Shanghai")
				}
			},
		},
		{
			name: "空时区应默认为Local",
			params: model.CreateConnectionParams{
				Name:     "空时区",
				Host:     "127.0.0.1",
				Port:     3306,
				Username: "root",
				Password: "test",
				Timezone: "",
			},
			wantErr: false,
			checkFn: func(t *testing.T, conn *model.Connection) {
				if conn.Timezone != "Local" {
					t.Errorf("空时区应默认为 Local, 实际 %q", conn.Timezone)
				}
			},
		},
		{
			name: "空名称应报错",
			params: model.CreateConnectionParams{
				Name:     "",
				Host:     "127.0.0.1",
				Username: "root",
			},
			wantErr: true,
		},
		{
			name: "空主机应报错",
			params: model.CreateConnectionParams{
				Name:     "空主机",
				Host:     "",
				Username: "root",
			},
			wantErr: true,
		},
		{
			name: "空用户名应报错",
			params: model.CreateConnectionParams{
				Name:     "空用户名",
				Host:     "127.0.0.1",
				Username: "",
			},
			wantErr: true,
		},
		{
			name: "端口为0时默认3306",
			params: model.CreateConnectionParams{
				Name:     "默认端口",
				Host:     "127.0.0.1",
				Port:     0,
				Username: "root",
				Password: "test",
			},
			wantErr: false,
			checkFn: func(t *testing.T, conn *model.Connection) {
				if conn.Port != 3306 {
					t.Errorf("默认端口应为 3306, 实际 %d", conn.Port)
				}
				if conn.Timezone != "Local" {
					t.Errorf("未传时区应默认为 Local, 实际 %q", conn.Timezone)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := svc.Create(tt.params)
			if tt.wantErr {
				if err == nil {
					t.Error("期望返回错误，但没有")
				}
				return
			}
			if err != nil {
				t.Errorf("不期望错误，但返回了: %v", err)
				return
			}
			if conn == nil {
				t.Error("返回的连接不应为 nil")
				return
			}
			if conn.Name != tt.params.Name {
				t.Errorf("连接名称 = %q, 期望 %q", conn.Name, tt.params.Name)
			}
			if conn.ID == "" {
				t.Error("连接ID不应为空")
			}
			if tt.checkFn != nil {
				tt.checkFn(t, conn)
			}
		})
	}
}

func TestConnectionService_CRUD(t *testing.T) {
	svc := newTestConnectionService()

	conn, err := svc.Create(model.CreateConnectionParams{
		Name:     "CRUD测试",
		Host:     "192.168.1.1",
		Port:     3307,
		Username: "admin",
		Password: "secret",
		Database: "testdb",
		GroupID:  "group_1",
		Timezone: "UTC",
	})
	if err != nil {
		t.Fatalf("创建连接失败: %v", err)
	}
	if conn.Timezone != "UTC" {
		t.Errorf("Timezone = %q, want %q", conn.Timezone, "UTC")
	}

	connections, err := svc.List()
	if err != nil {
		t.Fatalf("获取连接列表失败: %v", err)
	}
	found := false
	for _, c := range connections {
		if c.ID == conn.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("创建的连接在列表中找不到")
	}

	updated, err := svc.Update(model.UpdateConnectionParams{
		ID:       conn.ID,
		Name:     "CRUD测试(已修改)",
		Host:     "10.0.0.1",
		Port:     3308,
		Username: "admin2",
		Password: "newsecret",
		Database: "newdb",
		Timezone: "Asia/Shanghai",
	})
	if err != nil {
		t.Fatalf("更新连接失败: %v", err)
	}
	if updated.Name != "CRUD测试(已修改)" {
		t.Errorf("更新后名称 = %q, 期望 %q", updated.Name, "CRUD测试(已修改)")
	}
	if updated.Host != "10.0.0.1" {
		t.Errorf("更新后主机 = %q, 期望 %q", updated.Host, "10.0.0.1")
	}
	if updated.Timezone != "Asia/Shanghai" {
		t.Errorf("更新后时区 = %q, 期望 %q", updated.Timezone, "Asia/Shanghai")
	}
	if updated.UpdatedAt.Equal(conn.UpdatedAt) {
		t.Error("更新时间应已更新但未更新")
	}

	// Update with empty timezone should default to Local
	updated2, err := svc.Update(model.UpdateConnectionParams{
		ID:       conn.ID,
		Name:     "CRUD测试(空时区)",
		Host:     "10.0.0.1",
		Port:     3308,
		Username: "admin2",
		Password: "newsecret",
		Database: "newdb",
		Timezone: "",
	})
	if err != nil {
		t.Fatalf("更新连接(空时区)失败: %v", err)
	}
	if updated2.Timezone != "Local" {
		t.Errorf("空时区更新后应默认为 Local, 实际 %q", updated2.Timezone)
	}

	if err := svc.Delete(conn.ID); err != nil {
		t.Fatalf("删除连接失败: %v", err)
	}
	connections, err = svc.List()
	if err != nil {
		t.Fatalf("获取连接列表失败: %v", err)
	}
	for _, c := range connections {
		if c.ID == conn.ID {
			t.Fatal("已删除的连接仍在列表中")
		}
	}
}

func TestConnectionService_Delete_NotFound(t *testing.T) {
	svc := newTestConnectionService()
	err := svc.Delete("不存在的ID")
	if err == nil {
		t.Error("删除不存在的连接应返回错误")
	}
}

func TestConnectionService_Update_NotFound(t *testing.T) {
	svc := newTestConnectionService()
	_, err := svc.Update(model.UpdateConnectionParams{
		ID:       "不存在的ID",
		Name:     "test",
		Host:     "127.0.0.1",
		Username: "root",
	})
	if err == nil {
		t.Error("更新不存在的连接应返回错误")
	}
}

func TestConnectionService_Group_CRUD(t *testing.T) {
	svc := newTestConnectionService()

	group, err := svc.CreateGroup(model.CreateGroupParams{
		Name: "生产环境",
	})
	if err != nil {
		t.Fatalf("创建分组失败: %v", err)
	}
	if group.Name != "生产环境" {
		t.Errorf("分组名称 = %q, 期望 %q", group.Name, "生产环境")
	}

	groups, err := svc.ListGroups()
	if err != nil {
		t.Fatalf("获取分组列表失败: %v", err)
	}
	found := false
	for _, g := range groups {
		if g.ID == group.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("创建的分组在列表中找不到")
	}

	conn, err := svc.Create(model.CreateConnectionParams{
		Name:     "组内连接",
		Host:     "127.0.0.1",
		Username: "root",
		GroupID:  group.ID,
	})
	if err != nil {
		t.Fatalf("创建连接失败: %v", err)
	}

	if err := svc.DeleteGroup(group.ID); err != nil {
		t.Fatalf("删除分组失败: %v", err)
	}

	connections, err := svc.List()
	if err != nil {
		t.Fatalf("获取连接列表失败: %v", err)
	}
	for _, c := range connections {
		if c.ID == conn.ID {
			if c.GroupID != "" {
				t.Errorf("删除分组后连接应变为未分组, 实际 groupId = %q", c.GroupID)
			}
			return
		}
	}
	t.Fatal("连接在分组删除后仍应在列表中")
}

func TestConnectionService_TestConnection_NotFound(t *testing.T) {
	svc := newTestConnectionService()
	result := svc.TestConnection("不存在的ID")
	if result.Success {
		t.Error("测试不存在的连接应返回失败")
	}
}

func TestConnectionService_GetDatabases_NotFound(t *testing.T) {
	svc := newTestConnectionService()
	_, err := svc.GetDatabases("不存在的ID")
	if err == nil {
		t.Error("对不存在的连接查询数据库应返回错误")
	}
}

func TestConnectionService_GetTables_NotFound(t *testing.T) {
	svc := newTestConnectionService()
	_, err := svc.GetTables("不存在的ID", "testdb")
	if err == nil {
		t.Error("对不存在的连接查询表应返回错误")
	}
}
