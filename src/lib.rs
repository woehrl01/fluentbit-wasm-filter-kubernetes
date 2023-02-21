// Import pure and fast JSON library written in Rust
use serde_json::Value;

use std::slice;
use std::os::raw::c_char;
use std::io::Write;
use regex::Regex;

#[no_mangle]
pub extern "C" fn rust_filter(_tag: *const c_char, _tag_len: u32, _time_sec: u32, _time_nsec: u32, record: *const c_char, record_len: u32) -> *const u8 {
  let slice_record: &[u8] = unsafe { slice::from_raw_parts(record as *mut u8, record_len as usize) };
  let v: Value = serde_json::from_slice(slice_record).unwrap();

  let is_keep = filter_log(&v);
  if !is_keep {
    return std::ptr::null();
  }

  let mut result: Vec<u8> = Vec::new();
  result.write(slice_record).expect("Unable to write to vec");
  return result.as_ptr();
}

fn filter_log(record: &Value) -> bool {
  let container_name = record["container_name"].as_str().unwrap_or_default();
  let namespace_name = record["namespace_name"].as_str().unwrap_or_default();
  let pod_name = record["pod_name"].as_str().unwrap_or_default();

  //todo: extract the pod_name from the record (strip the random string from the end)
  //todo: read the config file from the path specified in the environment variable 

  let filter_log = get_filter(container_name, namespace_name, pod_name);

  let log = record["log"].as_str().unwrap();

  let is_match = Regex::new(&filter_log).unwrap().is_match(log);
  return is_match;
}

fn get_filter(container_name: &str, namespace_name: &str, pod_name: &str) -> String {
  let config: Value = serde_json::from_str(include_str!("config.json")).unwrap();

  let mut filter_log = "";
  if filter_log == "" {
    filter_log = config[container_name][namespace_name][pod_name].as_str().unwrap_or_default();
  }
  if filter_log == "" {
    filter_log = config[container_name][namespace_name]["*"].as_str().unwrap_or_default();
  }
  if filter_log == "" {
    filter_log = config[container_name]["*"][pod_name].as_str().unwrap_or_default();
  }
  if filter_log == "" {
    filter_log = config[container_name]["*"]["*"].as_str().unwrap_or_default();
  }
  if filter_log == "" {
    filter_log = config["*"][namespace_name][pod_name].as_str().unwrap_or_default();
  }
  if filter_log == "" {
    filter_log = config["*"]["*"][pod_name].as_str().unwrap_or_default();
  }
  if filter_log == "" {
    filter_log = config["*"][namespace_name]["*"].as_str().unwrap_or_default();
  }
  if filter_log == "" {
    filter_log = config["*"]["*"]["*"].as_str().unwrap_or_default();
  }
  return filter_log.to_string();
}

#[cfg(test)]
mod tests {
  use serde_json::json;
  use test_case::test_case;

  #[test_case("container1",  "namespace1", "pod1", "test", false  ; "when wildcard is used and log does not match")]
  #[test_case("container1",  "namespace1", "pod1", "abc",  true  ; "when wildcard is used and log matches")]
  #[test_case("a",           "b",          "c",    "test", false  ; "when no match is found")]
  #[test_case("a",  "b", "c", "def",  true  ; "when exact match is found")]
  #[test_case("a",  "b", "c", "adefg",  true  ; "when exact match is found as a substring")]
  fn filter(container_name: &str, namespace_name: &str, pod_name: &str, log: &str, expected: bool) {
    let v = json!({
      "container_name": container_name,
      "namespace_name": namespace_name,
      "pod_name": pod_name,
      "log": log
    });

    let is_keep = super::filter_log(&v);
    assert_eq!(is_keep, expected);
  }
}
