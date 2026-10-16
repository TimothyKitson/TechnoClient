# Void Client – TODO

## Void Launcher (modpack builder) — in progress

A CurseForge-style launcher for building and playing modpacks, themed purple and black.
Lives in `launcher/`. Phase 1 is built and shipping; standalone mode is next.

Done:
- [x] Browse and search mods live from Modrinth (free API, no key)
- [x] Create modpacks and add mods to them, then press Play
- [x] Pick the mod version that matches the pack's Minecraft version and loader
- [x] Only offer mods compatible with the pack's version and loader (search is facet-filtered)
- [x] Automatically pull in required dependencies (e.g. Fabric API, Cloth Config)
- [x] Purple and black theme everywhere, real Void logo + Modrinth icons
- [x] Modpack cover images (default + choose your own)
- [x] First-run choice: Minecraft Launcher (default) or Standalone
- [x] Minecraft-Launcher mode: write profile + download mods with SHA-512 checks
- [x] Ships as a single `VoidLauncher.exe` (no admin), on getvoidclient.com
- [x] One-click **Void FPS Boost** pack: bundles the Void Client jar + the speed mods,
      so the launcher download includes the FPS booster

Still to do:
- [ ] **Standalone mode**: download Minecraft + Java + Fabric and launch the game itself,
      with Microsoft (Xbox) login. This is the big remaining piece.
- [ ] Native app window instead of Edge `--app` (tray icon, clean quit)
- [ ] Warn on declared mod incompatibilities, not just missing dependencies
- [ ] Update/remove mods in a pack with version pinning; export a pack as `.mrpack`
- [ ] CurseForge as a second source (needs an approved API key)
- [ ] Quilt / NeoForge loaders (only Fabric is wired up now)

Useful Modrinth endpoints:
- `GET /v2/search?query=...&facets=[["versions:26.3"],["categories:fabric"]]`
- `GET /v2/project/{slug}/version?game_versions=["26.3"]&loaders=["fabric"]`
- Version `dependencies[]` entries carry `dependency_type` (`required`, `optional`, `incompatible`, `embedded`)
