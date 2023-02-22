mod config;

use config::{ConfigFileConfiguration, Configuration};
use regex::Regex;
use serde_json::Value;
use std::io::Write;
use std::os::raw::c_char;
use std::slice;

#[no_mangle]
pub extern "C" fn rust_filter(
    _tag: *const c_char,
    _tag_len: u32,
    _time_sec: u32,
    _time_nsec: u32,
    record: *const c_char,
    record_len: u32,
) -> *const u8 {
    let string_record: &[u8] =
        unsafe { slice::from_raw_parts(record as *mut u8, record_len as usize) };
    let json_record: Value = serde_json::from_slice(string_record).unwrap();

    let file_config = ConfigFileConfiguration::new();

    return match filter_log(&json_record, &file_config) {
        true => keep_log(string_record),
        false => skip_log(),
    };
}

fn skip_log() -> *const u8 {
    // if we wan't to skip the log, we just return null
    return std::ptr::null();
}

fn keep_log(slice_record: &[u8]) -> *const u8 {
    // if we wan't to keep the log, we need to return the input how it was received
    // we still need to return a new string, so we copy the input to a new vector
    let mut result: Vec<u8> = Vec::new();
    result.write(slice_record).expect("Unable to write to vec");
    return result.as_ptr();
}

fn extract_pod_name(full_pod_name: &str) -> String {
    let re = Regex::new(r"^(.+?)-[^%-]{10}-[^%-]{5}$|^(.+?)-\d+$|^(.+?)-[^%-]{5}$").unwrap();

    match re.captures(full_pod_name) {
        None => return full_pod_name.to_string(),
        Some(caps) => {
            for i in 1..=3 {
                if !caps.get(i).is_none() {
                    return caps.get(i).unwrap().as_str().to_string();
                }
            }

            return full_pod_name.to_string();
        }
    }
}

fn filter_log(record: &Value, config_source: &dyn Configuration) -> bool {
    let container_name = record["container_name"].as_str().unwrap_or_default();
    let namespace_name = record["namespace_name"].as_str().unwrap_or_default();
    let full_pod_name = record["pod_name"].as_str().unwrap_or_default();
    let log = record["log"].as_str().unwrap();

    let pod_name = extract_pod_name(full_pod_name);

    return match get_filter(container_name, namespace_name, &pod_name, config_source) {
        None => true,                                               // no filter found, keep log
        Some(filter) => Regex::new(&filter).unwrap().is_match(log), // filter found, keep log if it matches
    };
}

fn get_filter(
    container_name: &str,
    namespace_name: &str,
    pod_name: &str,
    config_source: &dyn Configuration,
) -> Option<String> {
    let config: &Value = config_source.get_config();

    let precedence = [
        (container_name, namespace_name, pod_name),
        (container_name, namespace_name, "*"),
        (container_name, "*", pod_name),
        (container_name, "*", "*"),
        ("*", namespace_name, pod_name),
        ("*", "*", pod_name),
        ("*", namespace_name, "*"),
        ("*", "*", "*"),
    ];

    for (container, namespace, pod) in precedence.iter() {
        if let Some(filter) = config[*container][*namespace][*pod].as_str() {
            return Some(filter.to_string());
        }
    }
    return None;
}

#[cfg(test)]
mod tests {
    use serde_json::json;
    use test_case::test_case;

    use crate::config::InMemoryConfiguration;

    #[test_case("container1",  "namespace1", "pod1", "test", false  ; "when wildcard is used and log does not match")]
    #[test_case("container1",  "namespace1", "pod1", "abc",  true  ; "when wildcard is used and log matches")]
    #[test_case("a",           "b",          "c",    "test", false  ; "when no match is found")]
    #[test_case("a",  "b", "c", "def",  true  ; "when exact match is found")]
    #[test_case("a",  "b", "c", "adefg",  true  ; "when exact match is found as a substring")]
    #[test_case("a",  "b", "document-generation-6499cbb75b-65lmt", "xyz",  true  ; "when pod name is from a deployment")]
    #[test_case("a",  "b", "argocd-application-controller-0", "xyz",  true  ; "when pod name is from a statefulset")]
    #[test_case("a",  "b", "argocd-application-controller-d", "xyz",  false  ; "when pod name is invalid")]
    fn filter(
        container_name: &str,
        namespace_name: &str,
        pod_name: &str,
        log: &str,
        expected: bool,
    ) {
        let config_source = InMemoryConfiguration::new(json! {
          {
            "*": {
                "*": {
                    "*": "abc",
                    "argocd-application-controller": "xyz",
                    "document-generation": "xyz"
                }
            },
            "a": {
                "b": {
                    "c": "def"
                }
            }
          }
        });

        let v = json!({
          "container_name": container_name,
          "namespace_name": namespace_name,
          "pod_name": pod_name,
          "log": log
        });

        let is_keep = super::filter_log(&v, &config_source);
        assert_eq!(is_keep, expected);
    }

    #[test_case("argocd-application-controller-0", "argocd-application-controller"  ; "when pod name is from a statefulset")]
    #[test_case("argocd-application-controller-d", "argocd-application-controller-d"  ; "when pod name is invalid")]
    #[test_case("document-generation-6499cbb75b-65lmt", "document-generation"  ; "when pod name is from a deployment")]
    #[test_case("worker-12438-m76v7", "worker-12438"  ; "when pod name is from a job or daemonset")]
    fn extract_pod_name(full_pod_name: &str, expected: &str) {
        let actual = super::extract_pod_name(full_pod_name);
        assert_eq!(actual, expected);
    }
}
