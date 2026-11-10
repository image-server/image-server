# README sample images

`sample.jpg` is an original AI-generated image created for this project with the built-in imagegen tool. It is 1536 × 1024 pixels. The scene was chosen to make the difference between a proportional resize, a square crop, and a wide crop easy to see.

The JPEG and WebP files in this directory are actual Image Server outputs, generated from that original with the default quality of 85 unless the filename specifies a different quality. They are committed so the main README displays without a running server or an external image host.

| File | Dimensions |
| --- | --- |
| `sample.jpg` | 1536 × 1024 |
| `w480.jpg` | 480 × 320 |
| `x240.jpg` | 240 × 240 |
| `480x240.jpg` | 480 × 240 |
| `w480-q80.webp` | 480 × 320 |
| `c0.250_0.250_0.750_0.750-w480.jpg` | 480 × 320 |

## Regenerate the variants

Start Image Server with local storage as shown in the main README, then run from the repository root (requires curl and Python 3):

```sh
sh docs/images/regenerate.sh
# Or point it at another local port:
sh docs/images/regenerate.sh http://localhost:17000
```

The script uploads `sample.jpg` to the `demo` namespace and overwrites only the five generated variants in this directory. The original remains unchanged. Use an instance without required URL signatures for this walkthrough.

## Original generation prompt

```text
Use case: photorealistic-natural
Asset type: original sample photograph for an open-source image resizing server README
Primary request: Create one beautiful editorial photograph of a small orange camper van parked beside an alpine lake, with evergreen forest and rugged mountains reflected in turquoise water. Fine natural detail that stays attractive in thumbnails.
Composition/framing: landscape 1536x1024, camper van near center so a center square crop preserves the subject, expansive scenery on either side to make cropping versus proportional resizing easy to see. Entire van visible in the central half of the frame.
Lighting/mood: warm early morning sunlight, natural vivid colors, calm outdoor adventure.
Constraints: one single continuous photographic image, no collage, no borders, no text, no logos, no watermarks, no people.
```
