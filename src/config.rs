use serde_json::Value;
use std::fs::read_to_string;

pub trait Configuration {
    fn get_config(&self) -> &Value;
}

pub struct ConfigFileConfiguration {
    config: Value,
}

impl Configuration for ConfigFileConfiguration {
    fn get_config(&self) -> &Value {
        return &self.config;
    }
}

fn get_config_file_path() -> String {
    let config_path = std::env::var("CONFIG_PATH").unwrap_or_default();
    if config_path == "" {
        return "./config.json".to_string();
    }
    return config_path;
}

impl ConfigFileConfiguration {
    pub fn new() -> ConfigFileConfiguration {
        let config_path = get_config_file_path();
        let content_of_config_file = read_to_string(config_path).unwrap();
        let config: Value = serde_json::from_str(&content_of_config_file).unwrap();
        return ConfigFileConfiguration { config };
    }
}

pub struct InMemoryConfiguration {
    config: Value,
}

impl Configuration for InMemoryConfiguration {
    fn get_config(&self) -> &Value {
        return &self.config;
    }
}

impl InMemoryConfiguration {
    #[allow(dead_code)] // used in tests
    pub fn new(config: Value) -> InMemoryConfiguration {
        return InMemoryConfiguration { config };
    }
}
