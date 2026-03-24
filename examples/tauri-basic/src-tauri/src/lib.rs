use serde::Serialize;
use tauri::{Emitter, Manager};

#[cfg(desktop)]
use tauri::{
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
};

#[cfg(desktop)]
use tauri_plugin_global_shortcut::GlobalShortcutExt;

// ── IPC Commands ─────────────────────────────────────────────────────────────

#[tauri::command]
fn greet(name: &str) -> String {
    format!("Hello, {}! From Tauri.", name)
}

#[derive(Serialize)]
struct SystemInfo {
    os: String,
    arch: String,
    tauri_version: String,
    app_version: String,
    debug: bool,
}

#[tauri::command]
fn get_system_info() -> SystemInfo {
    SystemInfo {
        os: std::env::consts::OS.to_string(),
        arch: std::env::consts::ARCH.to_string(),
        tauri_version: tauri::VERSION.to_string(),
        app_version: env!("CARGO_PKG_VERSION").to_string(),
        debug: cfg!(debug_assertions),
    }
}

#[tauri::command]
fn log_from_frontend(level: &str, message: &str) {
    match level {
        "error" => log::error!("[frontend] {}", message),
        "warn" => log::warn!("[frontend] {}", message),
        "info" => log::info!("[frontend] {}", message),
        "debug" => log::debug!("[frontend] {}", message),
        _ => log::trace!("[frontend] {}", message),
    }
}

// ── App Setup ────────────────────────────────────────────────────────────────

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let mut builder = tauri::Builder::default();

    // Single instance — desktop only (must be registered first)
    #[cfg(desktop)]
    {
        builder = builder.plugin(tauri_plugin_single_instance::init(|app, _argv, _cwd| {
            // Focus existing window when a second instance is launched
            if let Some(window) = app.get_webview_window("main") {
                let _ = window.show();
                let _ = window.set_focus();
            }
        }));
    }

    builder = builder
        // Plugins
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_os::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_clipboard_manager::init())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_window_state::Builder::new().build())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_autostart::init(
            tauri_plugin_autostart::MacosLauncher::LaunchAgent,
            None,
        ))
        .plugin(
            tauri_plugin_log::Builder::new()
                .level(log::LevelFilter::Info)
                .build(),
        );

    // Global shortcuts — desktop only
    #[cfg(desktop)]
    {
        builder = builder.plugin(
            tauri_plugin_global_shortcut::Builder::new()
                .with_handler(|app, _shortcut, event| {
                    if event.state == tauri_plugin_global_shortcut::ShortcutState::Pressed {
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.show();
                            let _ = window.set_focus();
                        }
                    }
                })
                .build(),
        );
    }

    // WebDriver plugin — only included when built with `--features webdriver`
    #[cfg(feature = "webdriver")]
    {
        builder = builder.plugin(tauri_plugin_webdriver::init());
    }

    builder
        // Commands
        .invoke_handler(tauri::generate_handler![
            greet,
            get_system_info,
            log_from_frontend,
        ])
        // Setup: devtools + tray
        .setup(|app| {
            #[cfg(all(debug_assertions, desktop))]
            {
                app.get_webview_window("main").unwrap().open_devtools();
            }

            // System tray (desktop only)
            #[cfg(desktop)]
            {
                let show = MenuItem::with_id(app, "show", "Show Window", true, None::<&str>)?;
                let quit = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;
                let menu = Menu::with_items(app, &[&show, &quit])?;

                TrayIconBuilder::new()
                    .icon(app.default_window_icon().unwrap().clone())
                    .menu(&menu)
                    .tooltip("Tauri Basic")
                    .on_menu_event(|app, event| match event.id.as_ref() {
                        "show" => {
                            if let Some(window) = app.get_webview_window("main") {
                                let _ = window.show();
                                let _ = window.set_focus();
                            }
                        }
                        "quit" => {
                            app.exit(0);
                        }
                        _ => {}
                    })
                    .on_tray_icon_event(|tray, event| {
                        if let TrayIconEvent::Click {
                            button: MouseButton::Left,
                            button_state: MouseButtonState::Up,
                            ..
                        } = event
                        {
                            let app = tray.app_handle();
                            if let Some(window) = app.get_webview_window("main") {
                                let _ = window.show();
                                let _ = window.set_focus();
                            }
                        }
                    })
                    .build(app)?;
            }

            // Global shortcut: CmdOrCtrl+Shift+T toggles window (desktop only)
            // Non-fatal — if another app owns this shortcut, we log and continue
            #[cfg(desktop)]
            {
                if let Err(e) = app.global_shortcut().register("CmdOrCtrl+Shift+T") {
                    log::warn!("Could not register global shortcut: {e}");
                }
            }

            // Deep link: forward URLs to frontend via event system
            #[cfg(desktop)]
            {
                use tauri_plugin_deep_link::DeepLinkExt;
                if let Ok(Some(urls)) = app.deep_link().get_current() {
                    let url_strings: Vec<String> = urls.iter().map(|u| u.to_string()).collect();
                    log::info!("App opened via deep link: {:?}", url_strings);
                    let _ = app.emit("deep-link-received", url_strings);
                }
                let handle = app.handle().clone();
                app.deep_link().on_open_url(move |event| {
                    let url_strings: Vec<String> = event.urls().iter().map(|u| u.to_string()).collect();
                    log::info!("Deep link received: {:?}", url_strings);
                    let _ = handle.emit("deep-link-received", url_strings);
                });
            }

            // Emit a welcome event to the frontend
            app.emit("backend-event", "App initialized successfully!")?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application")
}
