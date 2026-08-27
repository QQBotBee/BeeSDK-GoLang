#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <wchar.h>
#include <plugin_config.h>

#ifndef __GNUC__
#define __attribute__(x)
#endif

#define BEE_EXPORT
#define BEE_CALL __stdcall
#define IPC_TIMEOUT_MS 15000
#define STOP_TIMEOUT_MS 3000
#define MAX_IPC_LINE (8 * 1024 * 1024)

static HMODULE g_module;
static CRITICAL_SECTION g_lock;
static BOOL g_lock_ready;
static HANDLE g_process;
static HANDLE g_stdin_write;
static HANDLE g_stdout_read;
static DWORD g_request_id;
static WCHAR g_worker_path[MAX_PATH * 4];

static BOOL write_all(HANDLE handle, const void *data, DWORD size) {
    const BYTE *cursor = (const BYTE *)data;
    while (size) {
        DWORD written = 0;
        if (!WriteFile(handle, cursor, size, &written, NULL) || written == 0) return FALSE;
        cursor += written;
        size -= written;
    }
    return TRUE;
}

static char *base64_encode(const unsigned char *data, size_t len) {
    static const char table[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    size_t out_len = 4 * ((len + 2) / 3), i = 0, j = 0;
    char *out = (char *)HeapAlloc(GetProcessHeap(), 0, out_len + 1);
    if (!out) return NULL;
    while (i < len) {
        uint32_t a = i < len ? data[i++] : 0;
        uint32_t b = i < len ? data[i++] : 0;
        uint32_t c = i < len ? data[i++] : 0;
        uint32_t triple = (a << 16) | (b << 8) | c;
        out[j++] = table[(triple >> 18) & 63];
        out[j++] = table[(triple >> 12) & 63];
        out[j++] = table[(triple >> 6) & 63];
        out[j++] = table[triple & 63];
    }
    if (len % 3) out[out_len - 1] = '=';
    if (len % 3 == 1) out[out_len - 2] = '=';
    out[out_len] = 0;
    return out;
}

static char *json_escape(const char *value) {
    size_t len = value ? strlen(value) : 0, i, out_len = 0;
    char *out = (char *)HeapAlloc(GetProcessHeap(), 0, len * 6 + 1);
    if (!out) return NULL;
    for (i = 0; i < len; i++) {
        unsigned char ch = (unsigned char)value[i];
        if (ch == '"' || ch == '\\') { out[out_len++] = '\\'; out[out_len++] = (char)ch; }
        else if (ch == '\n') { out[out_len++] = '\\'; out[out_len++] = 'n'; }
        else if (ch == '\r') { out[out_len++] = '\\'; out[out_len++] = 'r'; }
        else if (ch == '\t') { out[out_len++] = '\\'; out[out_len++] = 't'; }
        else if (ch < 0x20) out_len += (size_t)sprintf(out + out_len, "\\u%04x", ch);
        else out[out_len++] = (char)ch;
    }
    out[out_len] = 0;
    return out;
}

static BOOL path_is_loader_subdir(const WCHAR *path) {
    const WCHAR *leaf = wcsrchr(path, L'\\');
    leaf = leaf ? leaf + 1 : path;
    return lstrcmpiW(leaf, L"plugin") == 0 || lstrcmpiW(leaf, L"temp_plugin") == 0;
}

static BOOL ensure_directory_tree(const WCHAR *path) {
    WCHAR copy[MAX_PATH * 4];
    WCHAR *p;
    if (lstrlenW(path) >= (int)(sizeof(copy) / sizeof(copy[0]))) return FALSE;
    lstrcpyW(copy, path);
    for (p = copy + 3; *p; p++) {
        if (*p == L'\\' || *p == L'/') {
            WCHAR saved = *p;
            *p = 0;
            CreateDirectoryW(copy, NULL);
            *p = saved;
        }
    }
    return CreateDirectoryW(copy, NULL) || GetLastError() == ERROR_ALREADY_EXISTS;
}

static BOOL build_worker_path(void) {
    WCHAR root[MAX_PATH * 4];
    WCHAR directory[MAX_PATH * 4];
    DWORD n = GetCurrentDirectoryW((DWORD)(sizeof(root) / sizeof(root[0])), root);
    WCHAR *slash;
    if (!n || n >= sizeof(root) / sizeof(root[0])) return FALSE;
    if (path_is_loader_subdir(root)) {
        slash = wcsrchr(root, L'\\');
        if (slash) *slash = 0;
    }
    if (swprintf(directory, sizeof(directory) / sizeof(directory[0]), L"%ls\\plugin_data\\%ls", root, BEE_PLUGIN_NAME_W) < 0) return FALSE;
    if (!ensure_directory_tree(directory)) return FALSE;
    return swprintf(g_worker_path, sizeof(g_worker_path) / sizeof(g_worker_path[0]), L"%ls\\%ls", directory, BEE_WORKER_FILENAME) >= 0;
}

static BOOL extract_worker(void) {
    HRSRC resource;
    HGLOBAL loaded;
    const void *bytes;
    DWORD size, written = 0;
    HANDLE file;
    if (!build_worker_path()) return FALSE;
    resource = FindResourceW(g_module, MAKEINTRESOURCEW(BEE_WORKER_RESOURCE_ID), MAKEINTRESOURCEW(10));
    if (!resource) return FALSE;
    loaded = LoadResource(g_module, resource);
    bytes = loaded ? LockResource(loaded) : NULL;
    size = SizeofResource(g_module, resource);
    if (!bytes || !size) return FALSE;
    file = CreateFileW(g_worker_path, GENERIC_WRITE, 0, NULL, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    if (file == INVALID_HANDLE_VALUE) return FALSE;
    if (!WriteFile(file, bytes, size, &written, NULL) || written != size) { CloseHandle(file); return FALSE; }
    FlushFileBuffers(file);
    CloseHandle(file);
    return TRUE;
}

static void close_handle(HANDLE *handle) {
    if (*handle) { CloseHandle(*handle); *handle = NULL; }
}

static BOOL worker_running(void) {
    return g_process && WaitForSingleObject(g_process, 0) == WAIT_TIMEOUT;
}

static BOOL start_worker(void) {
    SECURITY_ATTRIBUTES sa = {sizeof(sa), NULL, TRUE};
    HANDLE child_stdin_read = NULL, child_stdout_write = NULL;
    STARTUPINFOW si;
    PROCESS_INFORMATION pi;
    WCHAR command[MAX_PATH * 4 + 64];
    if (worker_running()) return TRUE;
    if (!extract_worker()) return FALSE;
    if (!CreatePipe(&child_stdin_read, &g_stdin_write, &sa, 0)) return FALSE;
    if (!SetHandleInformation(g_stdin_write, HANDLE_FLAG_INHERIT, 0)) goto fail;
    if (!CreatePipe(&g_stdout_read, &child_stdout_write, &sa, 0)) goto fail;
    if (!SetHandleInformation(g_stdout_read, HANDLE_FLAG_INHERIT, 0)) goto fail;
    ZeroMemory(&si, sizeof(si)); ZeroMemory(&pi, sizeof(pi));
    si.cb = sizeof(si); si.dwFlags = STARTF_USESTDHANDLES;
    si.hStdInput = child_stdin_read; si.hStdOutput = child_stdout_write; si.hStdError = GetStdHandle(STD_ERROR_HANDLE);
    swprintf(command, sizeof(command) / sizeof(command[0]), L"\"%ls\" --host-pid %lu", g_worker_path, GetCurrentProcessId());
    if (!CreateProcessW(g_worker_path, command, NULL, NULL, TRUE, CREATE_NO_WINDOW, NULL, NULL, &si, &pi)) goto fail;
    close_handle(&child_stdin_read); close_handle(&child_stdout_write); close_handle(&pi.hThread);
    g_process = pi.hProcess;
    return TRUE;
fail:
    close_handle(&child_stdin_read); close_handle(&child_stdout_write);
    close_handle(&g_stdin_write); close_handle(&g_stdout_read);
    return FALSE;
}

static BOOL read_line(char **line_out) {
    size_t capacity = 4096, length = 0;
    char *line = (char *)HeapAlloc(GetProcessHeap(), 0, capacity);
    if (!line) return FALSE;
    for (;;) {
        char ch; DWORD got = 0;
        if (!ReadFile(g_stdout_read, &ch, 1, &got, NULL) || got != 1) { HeapFree(GetProcessHeap(), 0, line); return FALSE; }
        if (ch == '\n') break;
        if (length + 1 >= capacity) {
            char *larger;
            if (capacity >= MAX_IPC_LINE) { HeapFree(GetProcessHeap(), 0, line); return FALSE; }
            capacity *= 2;
            larger = (char *)HeapReAlloc(GetProcessHeap(), 0, line, capacity);
            if (!larger) { HeapFree(GetProcessHeap(), 0, line); return FALSE; }
            line = larger;
        }
        if (ch != '\r') line[length++] = ch;
    }
    line[length] = 0; *line_out = line; return TRUE;
}

static const char *json_string_field(const char *json, const char *field, char **owned) {
    char pattern[64];
    const char *start, *end;
    size_t length;
    sprintf(pattern, "\"%s\":\"", field);
    start = strstr(json, pattern);
    if (!start) return NULL;
    start += strlen(pattern);
    end = strchr(start, '"');
    if (!end) return NULL;
    length = (size_t)(end - start);
    *owned = (char *)HeapAlloc(GetProcessHeap(), 0, length + 1);
    if (!*owned) return NULL;
    memcpy(*owned, start, length);
    (*owned)[length] = 0;
    return *owned;
}

static const char *json_string_or_number_field(const char *json, const char *field, char **owned) {
    char pattern[64];
    const char *start, *end;
    size_t length;
    sprintf(pattern, "\"%s\":", field);
    start = strstr(json, pattern);
    if (!start) return NULL;
    start += strlen(pattern);
    while (*start == ' ' || *start == '\t' || *start == '\r' || *start == '\n') start++;
    if (*start == '"') {
        start++;
        end = strchr(start, '"');
    } else {
        end = start;
        while ((*end >= '0' && *end <= '9') || *end == '-') end++;
    }
    if (!end || end == start) return NULL;
    length = (size_t)(end - start);
    *owned = (char *)HeapAlloc(GetProcessHeap(), 0, length + 1);
    if (!*owned) return NULL;
    memcpy(*owned, start, length);
    (*owned)[length] = 0;
    return *owned;
}

static unsigned char *base64_decode(const char *text, DWORD *size_out) {
    static signed char map[256];
    static BOOL initialized;
    size_t len, i, out_len = 0;
    unsigned char *out;
    int value = 0, bits = -8;
    if (!initialized) {
        memset(map, -1, sizeof(map));
        for (i = 0; i < 26; i++) { map['A' + i] = (signed char)i; map['a' + i] = (signed char)(26 + i); }
        for (i = 0; i < 10; i++) map['0' + i] = (signed char)(52 + i);
        map[(unsigned char)'+'] = 62; map[(unsigned char)'/'] = 63;
        initialized = TRUE;
    }
    len = text ? strlen(text) : 0;
    out = (unsigned char *)HeapAlloc(GetProcessHeap(), 0, len * 3 / 4 + 4);
    if (!out) return NULL;
    for (i = 0; i < len && text[i] != '='; i++) {
        int decoded = map[(unsigned char)text[i]];
        if (decoded < 0) continue;
        value = (value << 6) | decoded;
        bits += 6;
        if (bits >= 0) { out[out_len++] = (unsigned char)((value >> bits) & 0xff); bits -= 8; }
    }
    out[out_len] = 0;
    *size_out = (DWORD)out_len;
    return out;
}

typedef char *(BEE_CALL *BeeAPIFn)(char *command);

static BOOL utf8_command_to_gbk(const unsigned char *utf8, DWORD utf8_size, char **gbk_out) {
    int wide_len, gbk_len;
    WCHAR *wide;
    if (!utf8_size) {
        *gbk_out = (char *)HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, 1);
        return *gbk_out != NULL;
    }
    wide_len = MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, (const char *)utf8, (int)utf8_size, NULL, 0);
    if (!wide_len) return FALSE;
    wide = (WCHAR *)HeapAlloc(GetProcessHeap(), 0, (wide_len + 1) * sizeof(WCHAR));
    if (!wide) return FALSE;
    MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, (const char *)utf8, (int)utf8_size, wide, wide_len);
    wide[wide_len] = 0;
    gbk_len = WideCharToMultiByte(936, 0, wide, wide_len, NULL, 0, NULL, NULL);
    if (!gbk_len) { HeapFree(GetProcessHeap(), 0, wide); return FALSE; }
    *gbk_out = (char *)HeapAlloc(GetProcessHeap(), 0, gbk_len + 1);
    if (!*gbk_out) { HeapFree(GetProcessHeap(), 0, wide); return FALSE; }
    WideCharToMultiByte(936, 0, wide, wide_len, *gbk_out, gbk_len, NULL, NULL);
    (*gbk_out)[gbk_len] = 0;
    HeapFree(GetProcessHeap(), 0, wide);
    return TRUE;
}

static BOOL send_api_result(const char *id, const char *value_gbk, const char *error) {
    int wide_len = 0, utf8_len = 0;
    WCHAR *wide = NULL;
    char *utf8 = NULL, *encoded = NULL, *line = NULL, *escaped_error = NULL;
    BOOL ok = FALSE;
    if (value_gbk) {
        wide_len = MultiByteToWideChar(936, 0, value_gbk, -1, NULL, 0);
        if (wide_len > 0) {
            wide = (WCHAR *)HeapAlloc(GetProcessHeap(), 0, wide_len * sizeof(WCHAR));
            MultiByteToWideChar(936, 0, value_gbk, -1, wide, wide_len);
            utf8_len = WideCharToMultiByte(CP_UTF8, 0, wide, wide_len - 1, NULL, 0, NULL, NULL);
            utf8 = (char *)HeapAlloc(GetProcessHeap(), 0, utf8_len + 1);
            WideCharToMultiByte(CP_UTF8, 0, wide, wide_len - 1, utf8, utf8_len, NULL, NULL);
            utf8[utf8_len] = 0;
        }
    }
    encoded = base64_encode((const unsigned char *)(utf8 ? utf8 : ""), utf8 ? (size_t)utf8_len : 0);
    escaped_error = json_escape(error ? error : "");
    if (!encoded || !escaped_error) goto done;
    line = (char *)HeapAlloc(GetProcessHeap(), 0, strlen(id) + strlen(encoded) + strlen(escaped_error) + 96);
    if (!line) goto done;
    sprintf(line, "{\"type\":\"api_result\",\"id\":\"%s\",\"value_b64\":\"%s\",\"error\":\"%s\"}\n", id, encoded, escaped_error);
    ok = write_all(g_stdin_write, line, (DWORD)strlen(line));
done:
    if (wide) HeapFree(GetProcessHeap(), 0, wide);
    if (utf8) HeapFree(GetProcessHeap(), 0, utf8);
    if (encoded) HeapFree(GetProcessHeap(), 0, encoded);
    if (escaped_error) HeapFree(GetProcessHeap(), 0, escaped_error);
    if (line) HeapFree(GetProcessHeap(), 0, line);
    return ok;
}

static BOOL service_api_call(const char *line, const char *robot_json) {
    char *id = NULL, *command_b64 = NULL, *api_text = NULL, *gbk_command = NULL;
    unsigned char *utf8_command = NULL;
    DWORD utf8_size = 0;
    unsigned long address;
    char *result;
    BOOL ok = FALSE;
    if (!json_string_field(line, "id", &id) || !json_string_field(line, "command_b64", &command_b64)) goto done;
    if (!json_string_or_number_field(robot_json ? robot_json : "", "api", &api_text)) {
        send_api_result(id, NULL, "robot JSON missing api"); goto done;
    }
    address = strtoul(api_text, NULL, 10);
    if (!address) { send_api_result(id, NULL, "invalid robot api address"); goto done; }
    utf8_command = base64_decode(command_b64, &utf8_size);
    if (!utf8_command || !utf8_command_to_gbk(utf8_command, utf8_size, &gbk_command)) {
        send_api_result(id, NULL, "invalid api command"); goto done;
    }
    result = ((BeeAPIFn)(uintptr_t)address)(gbk_command);
    ok = send_api_result(id, result, NULL);
done:
    if (id) HeapFree(GetProcessHeap(), 0, id);
    if (command_b64) HeapFree(GetProcessHeap(), 0, command_b64);
    if (api_text) HeapFree(GetProcessHeap(), 0, api_text);
    if (utf8_command) HeapFree(GetProcessHeap(), 0, utf8_command);
    if (gbk_command) HeapFree(GetProcessHeap(), 0, gbk_command);
    return ok;
}

static int parse_result(const char *line) {
    const char *value = strstr(line, "\"result\":");
    return value ? atoi(value + 9) : 0;
}

static int send_event(const char *event, int argc, const char **argv) {
    char header[256], tail[] = "]}\n";
    DWORD id;
    int i, result = 0;
    char *line = NULL;
    const char *robot_json = argc > 0 ? argv[0] : NULL;
    if (!g_lock_ready) return 0;
    EnterCriticalSection(&g_lock);
    if (!start_worker()) goto done;
    id = ++g_request_id;
    sprintf(header, "{\"type\":\"event\",\"id\":\"%lu\",\"event\":\"%s\",\"args_b64\":[", id, event);
    if (!write_all(g_stdin_write, header, (DWORD)strlen(header))) goto done;
    for (i = 0; i < argc; i++) {
        const char *raw = argv[i] ? argv[i] : "";
        char *encoded = base64_encode((const unsigned char *)raw, strlen(raw));
        if (!encoded) goto done;
        if (i && !write_all(g_stdin_write, ",", 1)) { HeapFree(GetProcessHeap(), 0, encoded); goto done; }
        if (!write_all(g_stdin_write, "\"", 1) || !write_all(g_stdin_write, encoded, (DWORD)strlen(encoded)) || !write_all(g_stdin_write, "\"", 1)) {
            HeapFree(GetProcessHeap(), 0, encoded); goto done;
        }
        HeapFree(GetProcessHeap(), 0, encoded);
    }
    if (!write_all(g_stdin_write, tail, (DWORD)strlen(tail))) goto done;
    for (;;) {
        if (!read_line(&line)) goto done;
        if (strstr(line, "\"type\":\"api_call\"")) {
            if (!service_api_call(line, robot_json)) goto done;
            HeapFree(GetProcessHeap(), 0, line);
            line = NULL;
            continue;
        }
        break;
    }
    result = parse_result(line);
done:
    if (line) HeapFree(GetProcessHeap(), 0, line);
    LeaveCriticalSection(&g_lock);
    return result;
}

static void stop_worker(const char *lifecycle) {
    if (!g_lock_ready) return;
    if (worker_running()) send_event(lifecycle, 0, NULL);
    close_handle(&g_stdin_write);
    if (g_process && WaitForSingleObject(g_process, STOP_TIMEOUT_MS) == WAIT_TIMEOUT) TerminateProcess(g_process, 0);
    if (g_process) WaitForSingleObject(g_process, 1000);
    close_handle(&g_stdout_read); close_handle(&g_process);
}

BEE_EXPORT const char *BEE_CALL Bee_Init_Internal(const char *robot) {
	const char *a[] = {robot};
	extract_worker();
	send_event("initialize", 1, a);
	stop_worker("stop");
	return (const char *)BEE_INIT_JSON_GBK;
}
BEE_EXPORT void BEE_CALL Bee_Enable_Internal(const char *robot) { const char *a[] = {robot}; send_event("enable", 1, a); }
BEE_EXPORT void BEE_CALL Bee_Disable_Internal(const char *robot) { const char *a[] = {robot}; if (worker_running()) send_event("disable", 1, a); stop_worker("stop"); }
BEE_EXPORT void BEE_CALL Bee_Unload_Internal(const char *robot) { const char *a[] = {robot}; send_event("unload", 1, a); stop_worker("stop"); }
BEE_EXPORT void BEE_CALL Bee_Settings_Internal(const char *robot) { const char *a[] = {robot}; send_event("settings", 1, a); }
BEE_EXPORT int BEE_CALL Bee_ChannelPrivate_Internal(const char *r,const char *g,const char *c,const char *u,const char *m,const char *id) { const char *a[]={r,g,c,u,m,id}; return send_event("channel_private",6,a); }
BEE_EXPORT int BEE_CALL Bee_ChannelMessage_Internal(const char *r,const char *g,const char *c,const char *u,const char *m,const char *id) { const char *a[]={r,g,c,u,m,id}; return send_event("channel_message",6,a); }
BEE_EXPORT int BEE_CALL Bee_ChannelEvent_Internal(const char *r,const char *g,const char *c,const char *u,const char *o,const char *e,const char *m) { const char *a[]={r,g,c,u,o,e,m}; return send_event("channel_event",7,a); }
BEE_EXPORT int BEE_CALL Bee_PrivateMessage_Internal(const char *r,const char *u,const char *m,const char *id) { const char *a[]={r,u,m,id}; return send_event("private_message",4,a); }
BEE_EXPORT int BEE_CALL Bee_GroupMessage_Internal(const char *r,const char *g,const char *u,const char *m,const char *id) { const char *a[]={r,g,u,m,id}; return send_event("group_message",5,a); }
BEE_EXPORT int BEE_CALL Bee_CommonEvent_Internal(const char *r,const char *s,const char *u,const char *o,const char *e,const char *m) { const char *a[]={r,s,u,o,e,m}; return send_event("common_event",6,a); }

BOOL WINAPI DllMain(HINSTANCE instance, DWORD reason, LPVOID reserved) {
    (void)reserved;
    if (reason == DLL_PROCESS_ATTACH) {
        g_module = instance;
        InitializeCriticalSection(&g_lock);
        g_lock_ready = TRUE;
        DisableThreadLibraryCalls(instance);
    } else if (reason == DLL_PROCESS_DETACH && g_lock_ready) {
        /* Do not wait under the loader lock. Normal cleanup is synchronous in Bee_Unload_Internal. */
        close_handle(&g_stdin_write);
        close_handle(&g_stdout_read);
        close_handle(&g_process);
        DeleteCriticalSection(&g_lock);
        g_lock_ready = FALSE;
    }
    return TRUE;
}
