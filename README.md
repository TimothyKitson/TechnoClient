# Void Client

A free Minecraft 26.3 client that boosts FPS on weak laptops, with a purple-and-black Lunar-style HUD, zoom and Simple Voice Chat.

It works on any laptop. The first time it starts, Void Client checks your graphics card, RAM and CPU and picks the right settings by itself:

| Preset | Picked for | Render distance | FPS cap |
| --- | --- | --- | --- |
| Potato | Built-in graphics with under 7 GB RAM (for example Dell Latitude 3520 with 4 GB) | 6 | 60 |
| Balanced | Built-in graphics with 8 GB+ RAM, or a graphics card with under 12 GB RAM | 10 | 120 |
| High | NVIDIA / AMD graphics card with 12 GB+ RAM (for example Dell G16 7630) | 16 | Unlimited |

You can change the preset any time in the Void Client menu (**Right Shift**).

![Void Client HUD with FPS, CPS, coordinates and keystrokes](docs/hud.png)

![Void Client menu](docs/menu.png)

## What's inside

| Mod | What it does |
| --- | --- |
| Void Client | Auto-tuned settings, HUD (FPS, CPS, ping, coordinates, keystrokes), zoom, Right Shift menu |
| Sodium | Much faster chunk rendering (the big FPS boost) |
| Lithium | Faster game logic |
| FerriteCore | Uses a lot less RAM (fixes the "0 FPS for a few seconds" freezes) |
| ImmediatelyFast | Faster HUD, text and entity drawing |
| Entity Culling, MoreCulling | Skips drawing things you can't see |
| Dynamic FPS | Drops FPS when Minecraft is in the background so your laptop stays cool |
| Simple Voice Chat | Proximity voice chat on servers that have it installed |
| Fabric API, Cloth Config | Libraries the mods above need |

## Will this get me banned?

It shouldn't. Everything here is performance, visuals you already have, or info about yourself, the same kind of thing big servers like Hypixel allow:

- No X-ray, reach, autoclicker, kill aura, fullbright, freecam or minimap radar
- Void Client never sends anything to servers
- Coordinates hide themselves on servers that turn on "reduced debug info"
- Zoom, keystrokes, CPS and FPS displays are the same kind of features Lunar and OptiFine have

Every server makes its own rules, so if a server lists banned mods, give it a quick check.

## Install

Void Client uses **Prism Launcher**. It's free, and the portable version runs without admin rights, so it works on school laptops.

1. Go to [prismlauncher.org](https://prismlauncher.org/download/windows/), choose the **Portable** version, unzip it into a folder (for example `Documents\PrismLauncher`) and run `prismlauncher.exe`.
2. Sign in with your Microsoft account (Accounts, top right).
3. Download [`download/VoidClient.mrpack`](download/VoidClient.mrpack) from this repo (open it, then click the download button).
4. In Prism click **Add Instance → Import**, pick `VoidClient.mrpack`, then **OK**. Prism downloads all the mods and the right Java for you.
5. Set the RAM: right-click the instance → **Edit → Settings → Java**, tick **Memory** and set **Maximum memory allocation**:
   - **4 GB laptop (school Latitude 3520): 2048 MiB.** Giving it more makes the freezes worse, because Windows runs out of memory.
   - 8 GB laptop: 3072 MiB
   - 16 GB laptop (home G16 7630): 4096 MiB

   If this is set wrong, Void Client tells you the right number in a pop-up when the game starts.
6. Press **Launch**.

### Extra tips for the school laptop

- Plug in the charger. Laptops slow themselves down on battery.
- Windows Settings → System → Power → Power mode → **Best performance**.
- Close Chrome and other apps before playing. With 4 GB, every bit of RAM counts.
- In Prism's settings, turn on the option to close the launcher once the game opens, so it isn't using RAM while you play.

### Extra tip for the home laptop

Gaming laptops sometimes run Minecraft on the weak built-in Intel graphics instead of the NVIDIA card. If Void Client shows "Running on built-in graphics":

Windows Settings → System → Display → Graphics → **Add desktop app** → open the `java` folder inside your Prism folder → pick `javaw.exe` → **Options → High performance**. Then restart the game.

## Controls

| Key | Action |
| --- | --- |
| Right Shift | Void Client menu: turn HUD parts on or off, change preset |
| C (hold) | Zoom. Scroll while holding to zoom more or less |
| V | Simple Voice Chat menu |
| Caps Lock | Push to talk (Simple Voice Chat default) |

All keys can be changed in Options → Controls → Key Binds.

## For developers

Needs Java 25.

```
./gradlew build                  # builds build/libs/voidclient-<version>.jar
./gradlew runClient              # runs the game with Void Client and every mod in the pack
./gradlew runClientGameTest      # in-game automated test (world, HUD, zoom, CPS, menu)
python3 pack/build.py lock       # update pack/mods.lock.json to the newest 26.3 versions
python3 pack/build.py build --jar build/libs/voidclient-1.0.0.jar --version 1.0.0
```

The mod list lives in `pack/pack.json`, and the exact versions are pinned in `pack/mods.lock.json`.
