from PIL import Image, ImageDraw, ImageFilter, ImageChops
import numpy as np
import sys

S = 4096

def vertical_gradient(size, top, bottom):
    strip = Image.new("RGBA", (1, 256))
    for y in range(256):
        t = y / 255
        strip.putpixel((0, y), tuple(int(top[i] + (bottom[i] - top[i]) * t) for i in range(4)))
    return strip.resize(size, Image.BICUBIC)

def v_mask(bold):
    cx, cy = S / 2, S / 2
    if bold:
        top, arm, tip_y, notch_y, tip_half, spread = cy - 690, 520, cy + 790, cy + 250, 270, 880
    else:
        top, arm, tip_y, notch_y, tip_half, spread = cy - 600, 400, cy + 700, cy + 300, 200, 770
    points = [
        (cx - spread, top),
        (cx - spread + arm, top),
        (cx, notch_y),
        (cx + spread - arm, top),
        (cx + spread, top),
        (cx + tip_half, tip_y),
        (cx - tip_half, tip_y),
    ]
    mask = Image.new("L", (S, S), 0)
    ImageDraw.Draw(mask).polygon(points, fill=255)
    return mask

def render(bold):
    img = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    corner = Image.new("L", (S, S), 0)
    ImageDraw.Draw(corner).rounded_rectangle((0, 0, S - 1, S - 1), radius=int(S * (0.19 if bold else 0.225)), fill=255)
    bg = vertical_gradient((S, S), (36, 15, 68, 255) if bold else (26, 11, 48, 255), (8, 4, 16, 255) if bold else (5, 3, 10, 255))
    img.paste(bg, (0, 0), corner)

    cx = cy = S // 2
    r, w = (int(S * 0.39), int(S * 0.118)) if bold else (int(S * 0.325), int(S * 0.07))
    ring_box = (cx - r, cy - r, cx + r, cy + r)
    if not bold:
        glow = Image.new("RGBA", (S, S), (0, 0, 0, 0))
        ImageDraw.Draw(glow).ellipse(ring_box, outline=(139, 92, 246, 255), width=w + 240)
        glow = glow.filter(ImageFilter.GaussianBlur(220))
        img = Image.alpha_composite(img, Image.composite(glow, Image.new("RGBA", (S, S)), corner))

    ring = Image.new("L", (S, S), 0)
    ImageDraw.Draw(ring).ellipse(ring_box, fill=255)
    ImageDraw.Draw(ring).ellipse((cx - r + w, cy - r + w, cx + r - w, cy + r - w), fill=0)
    yy, xx = np.mgrid[0:S, 0:S].astype(np.float32)
    angle = np.arctan2(yy - cy, xx - cx)
    shine = np.clip(np.cos(angle - np.deg2rad(-125)), 0, 1) ** 2.2
    base = np.array([124, 77, 236], np.float32)
    light = np.array([226, 216, 255], np.float32)
    colors = base + (light - base) * shine[..., None]
    ring_fill = Image.fromarray(np.dstack([colors, np.full((S, S, 1), 255, np.float32)]).astype(np.uint8), "RGBA")
    img.paste(ring_fill, (0, 0), ring)

    hole = Image.new("L", (S, S), 0)
    ImageDraw.Draw(hole).ellipse((cx - r + w, cy - r + w, cx + r - w, cy + r - w), fill=255)
    img.paste(vertical_gradient((S, S), (14, 7, 28, 255), (2, 1, 5, 255)), (0, 0), hole)

    mask = v_mask(bold)
    if not bold:
        shadow = mask.filter(ImageFilter.GaussianBlur(90))
        img.paste(Image.new("RGBA", (S, S), (139, 92, 246, 255)), (0, 0), shadow.point(lambda v: int(v * 0.55)))
    img.paste(vertical_gradient((S, S), (255, 255, 255, 255), (206, 192, 255, 255)), (0, 0), mask)
    return img

out = sys.argv[1]
full, small = render(False), render(True)
for size in (1024, 512, 256, 128, 96, 64, 48):
    full.resize((size, size), Image.LANCZOS).save(f"{out}/icon{size}.png")
for size in (40, 32, 24, 20, 16):
    small.resize((size, size), Image.LANCZOS).save(f"{out}/icon{size}.png")
