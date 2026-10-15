package com.voidclient.test;

import com.mojang.blaze3d.platform.InputConstants;
import com.voidclient.Hardware;
import com.voidclient.Preset;
import com.voidclient.VoidClient;
import com.voidclient.Zoom;
import com.voidclient.gui.VoidScreen;
import net.fabricmc.fabric.api.client.gametest.v1.FabricClientGameTest;
import net.fabricmc.fabric.api.client.gametest.v1.context.ClientGameTestContext;
import net.fabricmc.fabric.api.client.gametest.v1.context.TestSingleplayerContext;
import net.minecraft.client.CloudStatus;
import net.minecraft.client.gui.components.AbstractWidget;
import net.minecraft.client.gui.components.events.GuiEventListener;

public class VoidClientGameTest implements FabricClientGameTest {
	@Override
	public void runTest(ClientGameTestContext context) {
		Hardware hardware = context.computeOnClient(minecraft -> VoidClient.hardware());
		VoidClient.LOGGER.info("[gametest] {} -> recommended {}", hardware, hardware.recommendedPreset());

		context.runOnClient(Preset.POTATO::apply);
		context.runOnClient(minecraft -> {
			check(minecraft.options.renderDistance().get() == 6, "potato render distance");
			check(minecraft.options.framerateLimit().get() == 60, "potato fps cap");
			check(minecraft.options.cloudStatus().get() == CloudStatus.OFF, "potato clouds");
		});

		try (TestSingleplayerContext world = context.worldBuilder().create()) {
			world.getConnection().waitForChunksRender();
			context.takeScreenshot("void-hud");

			context.runOnClient(minecraft -> VoidClient.config.smoothZoom = false);
			float normalFov = context.computeOnClient(minecraft -> minecraft.gameRenderer.mainCamera().getFov());
			context.getInput().holdKey(InputConstants.KEY_C);
			context.waitTicks(5);
			check(context.computeOnClient(minecraft -> Zoom.isActive()), "zoom active while C held");
			float zoomedFov = context.computeOnClient(minecraft -> minecraft.gameRenderer.mainCamera().getFov());
			check(zoomedFov < normalFov / 3, "zoom narrows fov " + normalFov + " -> " + zoomedFov);
			context.takeScreenshot("void-zoom");
			context.getInput().scroll(3);
			context.waitTicks(5);
			float scrolledFov = context.computeOnClient(minecraft -> minecraft.gameRenderer.mainCamera().getFov());
			check(scrolledFov < zoomedFov, "scroll zooms further " + zoomedFov + " -> " + scrolledFov);
			context.getInput().releaseKey(InputConstants.KEY_C);
			context.waitTicks(5);
			float restoredFov = context.computeOnClient(minecraft -> minecraft.gameRenderer.mainCamera().getFov());
			check(Math.abs(restoredFov - normalFov) < 0.5f, "fov restored " + normalFov + " vs " + restoredFov);
			context.runOnClient(minecraft -> VoidClient.config.smoothZoom = true);

			context.getInput().holdKey(options -> options.keyUp);
			for (int i = 0; i < 6; i++) {
				context.getInput().pressMouse(InputConstants.MOUSE_BUTTON_LEFT);
			}
			context.getInput().pressMouse(InputConstants.MOUSE_BUTTON_RIGHT);
			int leftCps = context.computeOnClient(minecraft -> VoidClient.LEFT_CLICKS.perSecond());
			int rightCps = context.computeOnClient(minecraft -> VoidClient.RIGHT_CLICKS.perSecond());
			check(leftCps >= 6 && rightCps >= 1, "cps counted left=" + leftCps + " right=" + rightCps);
			context.takeScreenshot("void-keystrokes");
			context.getInput().releaseKey(options -> options.keyUp);

			context.getInput().pressKey(InputConstants.KEY_RSHIFT);
			context.waitForScreen(VoidScreen.class);
			context.takeScreenshot("void-menu");
			clickButton(context, "FPS: ON");
			check(!context.computeOnClient(minecraft -> VoidClient.config.showFps), "fps toggle turned off");
			clickButton(context, "FPS: OFF");
			check(context.computeOnClient(minecraft -> VoidClient.config.showFps), "fps toggle turned back on");
			clickButton(context, "Done");
			context.waitForScreen(null);
		}
	}

	private static void clickButton(ClientGameTestContext context, String label) {
		double[] position = context.computeOnClient(minecraft -> {
			for (GuiEventListener child : minecraft.gui.screen().children()) {
				if (child instanceof AbstractWidget widget && widget.getMessage().getString().equals(label)) {
					double scale = minecraft.getWindow().getGuiScale();
					return new double[] {(widget.getX() + widget.getWidth() / 2.0) * scale, (widget.getY() + widget.getHeight() / 2.0) * scale};
				}
			}
			throw new AssertionError("No button labelled " + label);
		});
		context.getInput().setCursorPos(position[0], position[1]);
		context.getInput().pressMouse(InputConstants.MOUSE_BUTTON_LEFT);
		context.waitTick();
	}

	private static void check(boolean condition, String what) {
		if (!condition) {
			throw new AssertionError("Check failed: " + what);
		}
		VoidClient.LOGGER.info("[gametest] ok: {}", what);
	}
}
