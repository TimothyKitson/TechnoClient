# Void Client – TODO

## Next big feature: Void Launcher (modpack builder)

A CurseForge-style launcher for building and playing modpacks, themed purple and black like the rest of Void Client.

- [ ] Browse and search every mod (Modrinth first, its API is free and needs no key; CurseForge later, its API needs an approved key)
- [ ] Create modpacks and add mods to them, then press Play
- [ ] When adding a mod, pre-select the mod version that matches the pack's Minecraft version and loader
- [ ] Before downloading, check that the mod version is compatible with:
  - [ ] the pack's Minecraft version and loader
  - [ ] the other mods already in the pack (required dependencies, declared incompatibilities)
- [ ] If the mod works with neither the pack's version nor its other mods, offer to make a new pack with the right version and loader already filled in, so the only step left is pressing **Create**
- [ ] Automatically pull in required dependencies (for example Fabric API, Cloth Config)
- [ ] Purple and black theme everywhere (launcher, menus, pack pages)

Useful Modrinth endpoints:
- `GET /v2/search?query=...&facets=[["versions:26.3"],["categories:fabric"]]`
- `GET /v2/project/{slug}/version?game_versions=["26.3"]&loaders=["fabric"]`
- Version `dependencies[]` entries carry `dependency_type` (`required`, `optional`, `incompatible`, `embedded`)
