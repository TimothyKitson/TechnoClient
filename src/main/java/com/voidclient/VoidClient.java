package com.voidclient;

import com.mojang.blaze3d.platform.InputConstants;
import com.voidclient.gui.VoidHud;
import com.voidclient.gui.VoidScreen;
import net.fabricmc.api.ClientModInitializer;
import net.fabricmc.fabric.api.client.event.lifecycle.v1.ClientLifecycleEvents;
import net.fabricmc.fabric.api.client.event.lifecycle.v1.ClientTickEvents;
import net.fabricmc.fabric.api.client.keymapping.v1.KeyMappingHelper;
import net.fabricmc.fabric.api.client.rendering.v1.hud.HudElementRegistry;
import net.fabricmc.fabric.api.client.rendering.v1.hud.VanillaHudElements;
import net.minecraft.client.KeyMapping;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.components.toasts.SystemToast;
import net.minecraft.network.chat.Component;
import net.minecraft.resources.Identifier;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class VoidClient implements ClientModInitializer {
	public static final String MOD_ID = "voidclient";
	public static final Logger LOGGER = LoggerFactory.getLogger("VoidClient");
	public static final ClickCounter LEFT_CLICKS = new ClickCounter();
	public static final ClickCounter RIGHT_CLICKS = new ClickCounter();
	private static final long RAM_TOLERANCE_MB = 768;

	public static VoidConfig config = new VoidConfig();
	private static Hardware hardware;
	private static KeyMapping zoomKey;
	private static KeyMapping menuKey;

	@Override
	public void onInitializeClient() {
		config = VoidConfig.load();
		KeyMapping.Category category = KeyMapping.Category.register(Identifier.fromNamespaceAndPath(MOD_ID, "main"));
		zoomKey = KeyMappingHelper.registerKeyMapping(new KeyMapping("key.voidclient.zoom", InputConstants.KEY_C, category));
		menuKey = KeyMappingHelper.registerKeyMapping(new KeyMapping("key.voidclient.menu", InputConstants.KEY_RSHIFT, category));
		HudElementRegistry.attachElementBefore(VanillaHudElements.CHAT, Identifier.fromNamespaceAndPath(MOD_ID, "hud"), new VoidHud());
		ClientTickEvents.END_CLIENT_TICK.register(VoidClient::tick);
		ClientLifecycleEvents.CLIENT_STARTED.register(VoidClient::started);
	}

	public static Hardware hardware() {
		if (hardware == null) {
			hardware = Hardware.detect();
		}
		return hardware;
	}

	public static void onClick(int button) {
		if (button == InputConstants.MOUSE_BUTTON_LEFT) {
			LEFT_CLICKS.click();
		} else if (button == InputConstants.MOUSE_BUTTON_RIGHT) {
			RIGHT_CLICKS.click();
		}
	}

	public static Preset applyPreset(Minecraft minecraft) {
		Preset preset = Preset.byName(config.preset);
		if (preset == null) {
			preset = hardware().recommendedPreset();
		}
		preset.apply(minecraft);
		config.lastAppliedTier = preset.name();
		config.save();
		return preset;
	}

	private static void tick(Minecraft minecraft) {
		Zoom.setActive(zoomKey.isDown() && minecraft.gui.screen() == null);
		while (menuKey.consumeClick()) {
			if (minecraft.gui.screen() == null) {
				minecraft.gui.setScreen(new VoidScreen(null));
			}
		}
	}

	private static void started(Minecraft minecraft) {
		hardware = Hardware.detect();
		Hardware info = hardware;
		LOGGER.info("Detected {} ({}), {} MB RAM, {} threads, {} MB heap", info.gpuName(), info.gpuType(), info.totalRamMb(), info.cores(), info.maxHeapMb());

		if ("AUTO".equals(config.preset)) {
			Preset recommended = info.recommendedPreset();
			if (!recommended.name().equals(config.lastAppliedTier)) {
				Preset applied = applyPreset(minecraft);
				toast(minecraft, "Void Client: " + applied.label + " preset", "Tuned for this PC. Right Shift = menu");
			}
		}

		long recommendedHeap = info.recommendedHeapMb();
		if (info.maxHeapMb() > recommendedHeap + RAM_TOLERANCE_MB) {
			toast(minecraft, "Minecraft has too much RAM", "Set Max memory to " + recommendedHeap + " MB in the launcher");
		} else if (info.maxHeapMb() < recommendedHeap - RAM_TOLERANCE_MB) {
			toast(minecraft, "Minecraft could use more RAM", "Set Max memory to " + recommendedHeap + " MB in the launcher");
		}

		if (info.onIntegratedWhileNvidiaPresent()) {
			toast(minecraft, "Running on built-in graphics", "Set Java to NVIDIA in Windows Graphics settings");
		}
	}

	private static void toast(Minecraft minecraft, String title, String message) {
		SystemToast.add(minecraft.gui.toastManager(), new SystemToast.SystemToastId(10000L), Component.literal(title), Component.literal(message));
	}
}
