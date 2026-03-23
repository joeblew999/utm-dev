#[tauri::command]
fn greet(name: &str) -> String {
    format!("Hello, {}! From Tauri.", name)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![greet])
        .setup(|_app| {
            #[cfg(debug_assertions)]
            {
                use tauri::Manager;
                _app.get_webview_window("main").unwrap().open_devtools();
            }
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
