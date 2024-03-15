local _M = { _VERSION = '0.0.1' }

local SOCKS5_VERSION = 0x05
local NO_AUTH_METHOD = 0x00

local IPV4 = 0x01
local DOMAIN_NAME = 0x03
local IPV6 = 0x04

local CONNECT = 0x01
local BIND = 0x02
local UDP = 0x03

local SUCCEEDED = 0x00
local FAILURE = 0x01

-- 解析socks5 版本及支持的方法
local function resolve_method(socket)
    local data, err = socket:receive(2)
    if not data then
        ngx.exit(ngx.ERROR)
        return nil, err
    end

    local ver = string.byte(data, 1)
    local method_size = string.byte(data, 2)

    local methods, err = socket:receive(method_size)
    if not methods then
        ngx.exit(ngx.ERROR)
        return nil, err
    end

    return {
        ver = ver,
        methods = methods
    }, nil
end

-- 发送无需验证响应
local function send_no_auth_method(socket)
    -- socks5版本号+无需验证
    local data = string.char(SOCKS5_VERSION, NO_AUTH_METHOD)
    return socket:send(data)
end

local function resolve_dst_addr_port(socket)
    local data, err = socket:receive(4)
    if not data then
        ngx.log(ngx.ERR, "receive from socket failed, err: ", err)
        return nil, err
    end
    local ver = string.byte(data, 1)
    local cmd = string.byte(data, 2)
    local rsv = string.byte(data, 3)
    local atyp = string.byte(data, 4)
    local dst_len = 0
    if atyp == IPV4 then
        dst_len = 4
    else
        return nil, "unsupport atyp" .. atyp
    end

    local data, err = socket:receive(dst_len + 2)
    if err then
        ngx.log(ngx.ERR, "receive from socket failed, err: ", err)
        return nil, err
    end

    local dst = string.sub(data, 1, dst_len)
    local port_2 = string.byte(data, dst_len + 1)
    local port_1 = string.byte(data, dst_len + 2)
    local port = port_1 + port_2 * 256

    return {
        ver = ver,
        cmd = cmd,
        rsv = rsv,
        atyp = atyp,
        dst = dst,
        port = port
    }, nil
end

-- 返回响应
local function send_reply(socket, rep)
    local data = {}
    data[1] = string.char(SOCKS5_VERSION)
    data[2] = string.char(rep)
    data[3] = string.char(0x00)
    data[4] = string.char(IPV4)
    data[5] = "\x00\x00\x00\x00"
    data[6] = "\x00\x00"
    return socket:send(data)
end

function _M.run()
    -- 拿到请求socket
    local socket, err = assert(ngx.req.socket(true))
    if not socket then
        ngx.log(ngx.ERR, "get request sock failed, err: ", err)
        return ngx.exit(ngx.ERROR)
    end
    -- 解析method
    local ver_methods, err = resolve_method(socket)
    if err then
        ngx.log(ngx.ERR, "resolve method failed, err: ", err)
        return ngx.exit(ngx.ERROR)
    end
    -- 仅支持socks5
    if ver_methods.ver ~= SOCKS5_VERSION then
        ngx.log(ngx.ERR, "not support version", ver_methods.ver)
        return ngx.exit(ngx.ERROR)
    end

    -- 发送无需验证响应
    local ok, err = send_no_auth_method(socket)
    if err then
        ngx.log(ngx.ERR, "send no auth method failed, err: ", ver_methods.ver)
        return ngx.exit(ngx.ERROR)
    end

    -- 解析目的地址和端口
    local dst, err = resolve_dst_addr_port(socket)
    if err then
        ngx.log(ngx.ERR, "resolve , err: ", ver_methods.ver)
        return ngx.exit(ngx.ERROR)
    end

    local dst_addr = string.format("%d.%d.%d.%d",
            string.byte(dst.dst, 1),
            string.byte(dst.dst, 2),
            string.byte(dst.dst, 3),
            string.byte(dst.dst, 4))
    ngx.var.upstream = dst_addr .. ":" .. dst.port

    local ok, err = send_reply(socket, SUCCEEDED)
    if err then
        ngx.log(ngx.ERR, "send reply failed, err: ", ver_methods.ver)
        return ngx.exit(ngx.ERROR)
    end

end

return _M