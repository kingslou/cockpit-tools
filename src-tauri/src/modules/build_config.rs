/// feature_enabled 读取编译时嵌入的本地发行开关，缺失或解析失败时默认关闭。
pub(crate) fn feature_enabled(key: &str) -> bool {
    serde_json::from_str::<serde_json::Value>(include_str!("../../../build-config.json"))
        .ok()
        .and_then(|config| config.get(key).and_then(serde_json::Value::as_bool))
        .unwrap_or(false)
}
