use std::mem;
use std::os::raw::c_void;
use std::slice;
use std::str;
use std::ptr::copy;


/*
    Allocate a chunk of memory of `size` bytes in wasm module
*/
#[no_mangle]
pub extern "C" fn alloc(size: usize) -> *mut c_void {
    let mut buf = Vec::with_capacity(size);
    let ptr = buf.as_mut_ptr();
    mem::forget(buf);   // This is needed to ensure that memory is freed.
    return ptr as *mut c_void;
}

#[no_mangle]
pub extern fn fx(data_ptr: *mut c_void, size: u32) -> i32 {
    let slice = unsafe { slice::from_raw_parts(data_ptr as _, size as _) };
    let in_str = str::from_utf8(&slice).unwrap();
    let mut out_str = String::new();
    out_str += in_str;
    out_str += " <-- Tagged from Wasm";
    unsafe {
        copy(out_str.as_ptr(), data_ptr as *mut u8, out_str.len())
    };
    out_str.len() as i32
}