package com.voidclient;

import net.minecraft.client.CloudStatus;
import net.minecraft.client.GraphicsPreset;
import net.minecraft.client.Minecraft;
import net.minecraft.client.Options;
import net.minecraft.client.PrioritizeChunkUpdates;
import net.minecraft.server.level.ParticleStatus;

public enum Preset {
	POTATO("Potato"),
	BALANCED("Balanced"),
	HIGH("High");

	public final String label;

	Preset(String label) {
		this.label = label;
	}

	public void apply(Minecraft minecraft) {
		Options options = minecraft.options;
		switch (this) {
			case POTATO -> {
				GraphicsPreset.FAST.apply(minecraft);
				options.renderDistance().set(6);
				options.simulationDistance().set(5);
				options.biomeBlendRadius().set(0);
				options.ambientOcclusion().set(false);
				options.cloudStatus().set(CloudStatus.OFF);
				options.particles().set(ParticleStatus.MINIMAL);
				options.entityShadows().set(false);
				options.menuBackgroundBlurriness().set(0);
				options.prioritizeChunkUpdates().set(PrioritizeChunkUpdates.NONE);
				options.framerateLimit().set(60);
				options.enableVsync().set(false);
			}
			case BALANCED -> {
				GraphicsPreset.FAST.apply(minecraft);
				options.renderDistance().set(10);
				options.simulationDistance().set(6);
				options.ambientOcclusion().set(true);
				options.cloudStatus().set(CloudStatus.FAST);
				options.particles().set(ParticleStatus.DECREASED);
				options.entityShadows().set(false);
				options.framerateLimit().set(120);
				options.enableVsync().set(false);
			}
			case HIGH -> {
				GraphicsPreset.FANCY.apply(minecraft);
				options.renderDistance().set(16);
				options.simulationDistance().set(10);
				options.particles().set(ParticleStatus.ALL);
				options.cloudStatus().set(CloudStatus.FANCY);
				options.entityShadows().set(true);
				options.framerateLimit().set(260);
				options.enableVsync().set(false);
			}
		}
		options.graphicsPreset().set(GraphicsPreset.CUSTOM);
		options.save();
	}

	public static Preset byName(String name) {
		for (Preset preset : values()) {
			if (preset.name().equalsIgnoreCase(name)) {
				return preset;
			}
		}
		return null;
	}
}
