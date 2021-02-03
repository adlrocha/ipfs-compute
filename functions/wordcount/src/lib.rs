use std::mem;
use std::os::raw::c_void;
use std::slice;
use std::str;
use std::ptr::copy;
use std::collections::HashMap;
use serde_derive::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Debug)]
struct Counter {
    x: HashMap<String, u32>,
}

#[no_mangle]
pub extern "C" fn alloc(size: usize) -> *mut c_void {
    let mut buf = Vec::with_capacity(size);
    let ptr = buf.as_mut_ptr();
    mem::forget(buf);   // This is needed to ensure that memory is freed.
    return ptr as *mut c_void;
}


pub fn word_count(s: &str) -> HashMap<String, u32> {
    let mut counts = HashMap::new();

    for w in s.to_lowercase()
        .split(|c: char| !c.is_alphanumeric())
        .filter(|p| !p.is_empty()) {
        *counts.entry(w.to_owned()).or_insert(0u32) += 1u32;
    }

    counts
}

#[no_mangle]
pub extern fn fx(data_ptr: *mut c_void, size: u32) -> i32 {
    let slice = unsafe { slice::from_raw_parts(data_ptr as _, size as _) };
    let in_str = str::from_utf8(&slice).unwrap();
    // let a = String::from("text of words with words of a");
    let c = word_count(&in_str);
    let out = Counter{x: c};
    let out_str = serde_json::to_string(&out).unwrap();
    unsafe {
        copy(out_str.as_ptr(), data_ptr as *mut u8, out_str.len())
    };
    out_str.len() as i32
}
