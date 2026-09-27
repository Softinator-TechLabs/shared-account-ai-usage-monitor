#!/usr/bin/env python3
"""Render the product's four-module mark as a macOS app icon source PNG.
Requires Pillow only when regenerating the committed source art.
"""
from pathlib import Path
from PIL import Image, ImageDraw, ImageFilter

S = 1024
image = Image.new('RGBA', (S, S), (0, 0, 0, 0))
mask = Image.new('L', (S, S), 0)
ImageDraw.Draw(mask).rounded_rectangle((35, 35, 989, 989), radius=220, fill=255)
ground = Image.new('RGBA', (S, S))
p = ground.load()
for y in range(S):
    for x in range(S):
        t = min(1, (0.42*x + 0.82*y)/S)
        p[x, y] = (round(29-11*t), round(47-20*t), round(89-37*t), 255)
ground.putalpha(mask)
image.alpha_composite(ground)
# The brand's four modules are an intentional, legible 2x2 mark at menu sizes.
draw = ImageDraw.Draw(image)
white = (240, 246, 255, 255)
blue = (123, 174, 255, 255)
for box, color in [((235, 235, 459, 459), white), ((565, 565, 789, 789), white)]:
    draw.rounded_rectangle(box, radius=38, outline=color, width=43)
# Two open corners balance the closed frames.
draw.line([(617, 257), (775, 257), (775, 415)], fill=blue, width=47, joint='curve')
draw.line([(407, 767), (249, 767), (249, 609)], fill=blue, width=47, joint='curve')
for box in [(590, 232, 636, 282), (750, 390, 800, 440), (382, 742, 432, 792), (224, 584, 274, 634)]:
    draw.rounded_rectangle(box, radius=17, fill=blue)
out = Path(__file__).resolve().parents[1] / 'desktop/macos/assets/AppIcon-1024.png'
image.save(out, optimize=True)
menu = Image.new('RGBA', (128, 128), (0, 0, 0, 0))
m = ImageDraw.Draw(menu)
for box in [(25, 25, 51, 51), (77, 77, 103, 103)]:
    m.rounded_rectangle(box, radius=5, outline=(0, 0, 0, 255), width=7)
m.line([(80, 28), (101, 28), (101, 49)], fill=(0, 0, 0, 255), width=7)
m.line([(48, 100), (27, 100), (27, 79)], fill=(0, 0, 0, 255), width=7)
menu = menu.resize((64, 64), Image.Resampling.LANCZOS)
menu.save(out.with_name('MenuBarTemplate.png'), optimize=True)
print(out)
