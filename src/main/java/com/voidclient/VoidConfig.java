package com.voidclient;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import net.fabricmc.loader.api.FabricLoader;

import java.io.IOException;
import java.io.Reader;
import java.io.Writer;
import java.nio.file.Files;
import java.nio.file.Path;

public class VoidConfig {
	private static final Gson GSON = new GsonBuilder().setPrettyPrinting().create();
	private static final Path FILE = FabricLoader.getInstance().getConfigDir().resolve("voidclient.json");

	public String preset = "AUTO";
	public String lastAppliedTier = "";
	public boolean showFps = true;
	public boolean showCps = true;
	public boolean showPing = true;
	public boolean showCoords = true;
	public boolean showKeystrokes = true;
	public boolean smoothZoom = true;
	public double zoomFactor = 4.0;
	public boolean purpleOutline = true;

	public static VoidConfig load() {
		if (Files.exists(FILE)) {
			try (Reader reader = Files.newBufferedReader(FILE)) {
				VoidConfig config = GSON.fromJson(reader, VoidConfig.class);
				if (config != null) {
					return config;
				}
			} catch (IOException | RuntimeException e) {
				VoidClient.LOGGER.warn("Could not read {}, using defaults", FILE, e);
			}
		}
		return new VoidConfig();
	}

	public void save() {
		try {
			Files.createDirectories(FILE.getParent());
			try (Writer writer = Files.newBufferedWriter(FILE)) {
				GSON.toJson(this, writer);
			}
		} catch (IOException e) {
			VoidClient.LOGGER.warn("Could not save {}", FILE, e);
		}
	}
}
