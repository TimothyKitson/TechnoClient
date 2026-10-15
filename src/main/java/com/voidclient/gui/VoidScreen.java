package com.voidclient.gui;

import com.voidclient.Hardware;
import com.voidclient.Preset;
import com.voidclient.VoidClient;
import com.voidclient.VoidConfig;
import com.voidclient.Theme;
import net.minecraft.client.gui.GuiGraphicsExtractor;
import net.minecraft.client.gui.screens.Screen;
import net.minecraft.network.chat.Component;

import java.util.function.Consumer;
import java.util.function.Supplier;

public class VoidScreen extends Screen {
	private static final int BUTTON_WIDTH = 150;
	private static final int BUTTON_HEIGHT = 20;
	private static final String[] PRESET_CYCLE = {"AUTO", "POTATO", "BALANCED", "HIGH"};

	private final Screen parent;
	private String status = "";

	public VoidScreen(Screen parent) {
		super(Component.literal("VOID CLIENT"));
		this.parent = parent;
	}

	@Override
	protected void init() {
		VoidConfig config = VoidClient.config;
		int left = width / 2 - BUTTON_WIDTH - 5;
		int right = width / 2 + 5;
		int y = height / 6 + 28;

		toggle(left, y, "FPS", () -> config.showFps, value -> config.showFps = value);
		toggle(right, y, "CPS", () -> config.showCps, value -> config.showCps = value);
		y += 24;
		toggle(left, y, "Ping", () -> config.showPing, value -> config.showPing = value);
		toggle(right, y, "Coordinates", () -> config.showCoords, value -> config.showCoords = value);
		y += 24;
		toggle(left, y, "Keystrokes", () -> config.showKeystrokes, value -> config.showKeystrokes = value);
		toggle(right, y, "Smooth zoom", () -> config.smoothZoom, value -> config.smoothZoom = value);
		y += 36;

		addRenderableWidget(new PurpleButton(left, y, BUTTON_WIDTH, BUTTON_HEIGHT, presetLabel(), button -> {
			config.preset = next(config.preset);
			button.setMessage(presetLabel());
			config.save();
		}));
		addRenderableWidget(new PurpleButton(right, y, BUTTON_WIDTH, BUTTON_HEIGHT, Component.literal("Apply preset now"), button -> {
			Preset preset = VoidClient.applyPreset(minecraft);
			status = preset.label + " preset applied";
		}));

		addRenderableWidget(new PurpleButton(width / 2 - 100, height - 32, 200, BUTTON_HEIGHT, Component.literal("Done"), button -> onClose()));
	}

	private void toggle(int x, int y, String name, Supplier<Boolean> getter, Consumer<Boolean> setter) {
		addRenderableWidget(new PurpleButton(x, y, BUTTON_WIDTH, BUTTON_HEIGHT, toggleLabel(name, getter.get()), button -> {
			boolean value = !getter.get();
			setter.accept(value);
			button.setMessage(toggleLabel(name, value));
			VoidClient.config.save();
		}));
	}

	private static Component toggleLabel(String name, boolean value) {
		return Component.literal(name + ": " + (value ? "ON" : "OFF"));
	}

	private static Component presetLabel() {
		String preset = VoidClient.config.preset;
		if ("AUTO".equals(preset)) {
			return Component.literal("Preset: Auto (" + VoidClient.hardware().recommendedPreset().label + ")");
		}
		Preset chosen = Preset.byName(preset);
		return Component.literal("Preset: " + (chosen == null ? preset : chosen.label));
	}

	private static String next(String current) {
		for (int i = 0; i < PRESET_CYCLE.length; i++) {
			if (PRESET_CYCLE[i].equals(current)) {
				return PRESET_CYCLE[(i + 1) % PRESET_CYCLE.length];
			}
		}
		return PRESET_CYCLE[0];
	}

	@Override
	public void extractBackground(GuiGraphicsExtractor graphics, int mouseX, int mouseY, float partialTick) {
		graphics.fillGradient(0, 0, width, height, Theme.BACKGROUND_TOP, Theme.BACKGROUND_BOTTOM);
	}

	@Override
	public void extractRenderState(GuiGraphicsExtractor graphics, int mouseX, int mouseY, float partialTick) {
		super.extractRenderState(graphics, mouseX, mouseY, partialTick);
		int center = width / 2;
		int top = height / 6 - 14;
		graphics.centeredText(font, title, center, top, Theme.PURPLE_BRIGHT);
		graphics.fill(center - 40, top + 11, center + 40, top + 12, Theme.PURPLE);
		Hardware hardware = VoidClient.hardware();
		String specs = hardware.ramLabel() + "  |  " + hardware.cores() + " threads  |  Minecraft RAM " + hardware.maxHeapMb() + " MB";
		graphics.centeredText(font, font.plainSubstrByWidth(hardware.gpuName(), width - 20), center, top + 17, Theme.TEXT_MUTED);
		graphics.centeredText(font, specs, center, top + 28, Theme.TEXT_MUTED);
		graphics.centeredText(font, "Zoom: hold C, scroll to zoom more   |   Menu: Right Shift", center, height - 60, Theme.TEXT_MUTED);
		if (!status.isEmpty()) {
			graphics.centeredText(font, status, center, height - 46, Theme.PURPLE_BRIGHT);
		}
	}

	@Override
	public void onClose() {
		VoidClient.config.save();
		minecraft.gui.setScreen(parent);
	}
}
