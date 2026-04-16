import struct

def get_png_size(filename):
    try:
        with open(filename, 'rb') as f:
            data = f.read(24)
            if data[:8] == b'\x89PNG\r\n\x1a\n' and data[12:16] == b'IHDR':
                w, h = struct.unpack('>LL', data[16:24])
                return w, h
    except Exception as e:
        return str(e)
    return None

print(f"192x192 icon size: {get_png_size('frontend/public/pwa-192x192.png')}")
print(f"512x512 icon size: {get_png_size('frontend/public/pwa-512x512.png')}")
