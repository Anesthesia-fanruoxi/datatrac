// Tauri 应用入口
// 负责启动应用并初始化所有服务

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

fn main() {
    app_lib::run()
}
