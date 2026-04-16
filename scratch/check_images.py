from PIL import Image
try:
    with Image.open('frontend/public/pwa-192x192.png') as img:
        print(f'192 icon: {img.size}')
    with Image.open('frontend/public/pwa-512x512.png') as img:
        print(f'512 icon: {img.size}')
except Exception as e:
    print(f'Error: {e}')
