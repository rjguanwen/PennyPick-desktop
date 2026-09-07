# -*- coding: utf-8 -*-
"""Generate a branded app icon for PennyPick (ShiCai) desktop.

Design: rounded-square deep-blue gradient canvas + glossy golden coin
with the Chinese character "shi" (拾) embossed in white + golden sparkles.
Outputs multi-resolution .ico (16..256) and a preview .png.
"""
import os

from PIL import Image, ImageDraw, ImageFilter, ImageFont
import numpy as np

SIZE = 1024
OUT_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")
os.makedirs(OUT_DIR, exist_ok=True)

PNG_PATH = os.path.join(OUT_DIR, "pennypick_icon.png")
ICO_PATH = os.path.join(OUT_DIR, "pennypick_icon.ico")


def lerp(a, b, t):
    return a + (b - a) * t


def make_bg():
    """Vertical gradient rounded-square background (single smooth ramp)."""
    top = np.array([12, 58, 134], dtype=np.float64)     # #0C3A86
    bottom = np.array([86, 156, 255], dtype=np.float64)  # #569CFF
    img = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    arr = np.zeros((SIZE, SIZE, 4), dtype=np.uint8)
    ys = np.arange(SIZE).reshape(-1, 1).astype(np.float64)
    t = ys / (SIZE - 1)
    c = top + (bottom - top) * t
    arr[:, :, :3] = c.astype(np.uint8)
    arr[:, :, 3] = 255
    img = Image.fromarray(arr, "RGBA")
    mask = Image.new("L", (SIZE, SIZE), 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, SIZE - 1, SIZE - 1], radius=225, fill=255)
    # thin white border for glass edge
    glow = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    ImageDraw.Draw(glow).rounded_rectangle([0, 0, SIZE - 1, SIZE - 1], radius=225,
                                            outline=(255, 255, 255, 32), width=8)
    img = Image.alpha_composite(img, glow)
    img.putalpha(mask)
    # top light arc to give depth (very subtle)
    light = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    ImageDraw.Draw(light).pieslice([-260, -380, SIZE + 260, 880], 180, 360, fill=(255, 255, 255, 22))
    light = light.filter(ImageFilter.GaussianBlur(70))
    return Image.alpha_composite(img, light)


def make_coin():
    """Glossy golden coin layer with 拾 character, transparent background."""
    coin = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    cx, cy, r = 512, 566, 300
    # outer rim (darker gold)
    rim = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    rd = ImageDraw.Draw(rim)
    rd.ellipse([cx - r - 18, cy - r - 18, cx + r + 18, cy + r + 18], fill=(106, 74, 16, 255))
    coin = Image.alpha_composite(coin, rim)
    # inner coin face vertical gradient gold
    arr = np.zeros((SIZE, SIZE, 4), dtype=np.uint8)
    t_top = np.array([255, 232, 168], dtype=np.float64)   # #FFE8A8
    t_mid = np.array([255, 201, 77], dtype=np.float64)    # #FFC94D
    t_bot = np.array([229, 154, 31], dtype=np.float64)    # #E59A1F
    for y in range(SIZE):
        t = (y - (cy - r)) / (2 * r)
        t = max(0.0, min(1.0, t))
        if t < 0.5:
            tt = t / 0.5
            c = lerp(t_top, t_mid, tt)
        else:
            tt = (t - 0.5) / 0.5
            c = lerp(t_mid, t_bot, tt)
        arr[y, :, :3] = c.astype(np.uint8)
        arr[y, :, 3] = 255
    face = Image.fromarray(arr, "RGBA")
    mask = Image.new("L", (SIZE, SIZE), 0)
    ImageDraw.Draw(mask).ellipse([cx - r, cy - r, cx + r, cy + r], fill=255)
    face.putalpha(mask)
    coin = Image.alpha_composite(coin, face)
    # radial highlight top-left
    hl = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    hd = ImageDraw.Draw(hl)
    hd.ellipse([cx - int(0.6 * r), cy - int(1.07 * r),
                cx + int(0.75 * r), cy + int(0.16 * r)], fill=(255, 255, 255, 64))
    hl = hl.filter(ImageFilter.GaussianBlur(int(0.24 * r)))
    coin = Image.alpha_composite(coin, hl)
    # bottom-right subtle shadow arc inside coin
    sh = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    sd = ImageDraw.Draw(sh)
    sd.ellipse([cx - int(0.83 * r), cy + int(0.16 * r),
                cx + int(0.93 * r), cy + int(1.19 * r)], fill=(140, 88, 10, 90))
    sh = sh.filter(ImageFilter.GaussianBlur(int(0.22 * r)))
    coin = Image.alpha_composite(coin, sh)
    # embossed coin ring
    ring = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    rgd = ImageDraw.Draw(ring)
    rgd.ellipse([cx - r + 24, cy - r + 24, cx + r - 24, cy + r - 24], outline=(124, 84, 20, 90), width=6)
    ring = ring.filter(ImageFilter.GaussianBlur(2))
    coin = Image.alpha_composite(coin, ring)
    return coin, cx, cy


def pick_font():
    candidates = [
        r"C:\Windows\Fonts\msyhbd.ttc",   # Microsoft YaHei Bold
        r"C:\Windows\Fonts\msyh.ttc",
        r"C:\Windows\Fonts\simhei.ttf",
        r"C:\Windows\Fonts\arialbd.ttf",
    ]
    for p in candidates:
        if os.path.exists(p):
            return p
    return None


def draw_character(coin, cx, cy):
    """White 拾 glyph with dark-gold emboss on the coin.

    Drawn via dedicated layers + alpha masks so that anti-aliased edges
    blend properly instead of punching semi-transparent holes in the coin.
    """
    font_path = pick_font()
    font = ImageFont.truetype(font_path, 380) if font_path else ImageFont.load_default()
    txt = "\u62fe"  # 拾

    def text_mask(stroke, dx=0, dy=0):
        m = Image.new("L", (SIZE, SIZE), 0)
        md = ImageDraw.Draw(m)
        bbox = md.textbbox((0, 0), txt, font=font, stroke_width=stroke)
        w, h = bbox[2] - bbox[0], bbox[3] - bbox[1]
        x0 = cx - w / 2 - bbox[0] + dx
        y0 = cy - h / 2 - bbox[1] + dy
        md.text((x0, y0), txt, font=font, fill=255,
                stroke_width=stroke, stroke_fill=255)
        return m

    sh_mask = text_mask(20, dx=9, dy=12)
    sh_layer = Image.new("RGBA", (SIZE, SIZE), (148, 95, 16, 255))
    sh_layer.putalpha(sh_mask)
    coin = Image.alpha_composite(coin, sh_layer)

    wh_mask = text_mask(8)
    wh_layer = Image.new("RGBA", (SIZE, SIZE), (255, 255, 255, 255))
    wh_layer.putalpha(wh_mask)
    coin = Image.alpha_composite(coin, wh_layer)
    return coin


def make_sparkles():
    """Golden sparkle dots scattered in upper/outer area."""
    sp = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    d = ImageDraw.Draw(sp)
    pts = [
        (205, 250, 20), (812, 225, 15), (870, 470, 10),
        (160, 430, 12), (168, 668, 9), (836, 700, 12), (655, 175, 8),
    ]
    for px, py, pr in pts:
        # 4-point star via two rotated squares
        d.polygon([(px, py - pr * 3.2), (px + pr * 0.9, py - pr * 0.9),
                   (px + pr * 3.2, py), (px + pr * 0.9, py + pr * 0.9),
                   (px, py + pr * 3.2), (px - pr * 0.9, py + pr * 0.9),
                   (px - pr * 3.2, py), (px - pr * 0.9, py - pr * 0.9)],
                  fill=(255, 230, 150, 230))
    sp = sp.filter(ImageFilter.GaussianBlur(2))
    return sp


def main():
    bg = make_bg()
    coin, cx, cy = make_coin()
    coin = draw_character(coin, cx, cy - 8)
    sparkles = make_sparkles()
    icon = Image.alpha_composite(bg, sparkles)
    icon = Image.alpha_composite(icon, coin)

    icon.convert("RGBA").save(PNG_PATH)

    sizes = [256, 128, 64, 48, 32, 24, 16]
    base = icon.resize((256, 256), Image.LANCZOS).convert("RGBA")
    others = [icon.resize((s, s), Image.LANCZOS).convert("RGBA") for s in sizes[1:]]
    base.save(ICO_PATH, format="ICO",
              sizes=[(s, s) for s in sizes],
              append_images=others)
    print("OK:", PNG_PATH)
    print("OK:", ICO_PATH)


if __name__ == "__main__":
    main()
