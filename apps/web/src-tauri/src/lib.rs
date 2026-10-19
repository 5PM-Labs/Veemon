// Token commands. The token lives in tauri-plugin-store's `auth.json` in the
// OS app-data directory, never in web storage, so page script cannot read it by
// key. These are the only callers allowed to touch it.
const TOKEN_KEY: &str = "token";

fn store<R: tauri::Runtime>(app: &tauri::AppHandle<R>) -> Result<tauri_plugin_store::Store<R>, String> {
    tauri_plugin_store::StoreBuilder::new(app, "auth.json")
        .build()
        .map_err(|e| e.to_string())
}

#[tauri::command]
fn get_token<R: tauri::Runtime>(app: tauri::AppHandle<R>) -> Result<Option<String>, String> {
    Ok(store(&app)?
        .get(TOKEN_KEY)
        .and_then(|v| v.as_str().map(str::to_owned)))
}

#[tauri::command]
fn set_token<R: tauri::Runtime>(app: tauri::AppHandle<R>, token: String) -> Result<(), String> {
    let mut store = store(&app)?;
    store.insert(TOKEN_KEY.into(), serde_json::Value::String(token));
    store.save().map_err(|e| e.to_string())
}

#[tauri::command]
fn clear_token<R: tauri::Runtime>(app: tauri::AppHandle<R>) -> Result<(), String> {
    let mut store = store(&app)?;
    store.delete(TOKEN_KEY);
    store.save().map_err(|e| e.to_string())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_store::Builder::default().build())
        .invoke_handler(tauri::generate_handler![get_token, set_token, clear_token])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
