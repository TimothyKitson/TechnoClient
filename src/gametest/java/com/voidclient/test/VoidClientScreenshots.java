package com.voidclient.test;

import com.voidclient.Preset;
import com.voidclient.VoidClient;
import com.voidclient.gui.VoidScreen;
import net.fabricmc.fabric.api.client.gametest.v1.FabricClientGameTest;
import net.fabricmc.fabric.api.client.gametest.v1.context.ClientGameTestContext;
import net.fabricmc.fabric.api.client.gametest.v1.context.TestSingleplayerContext;
import net.minecraft.world.entity.player.ChatVisiblity;

public class VoidClientScreenshots implements FabricClientGameTest {
	private static final int RENDER_TIMEOUT = 20 * 60 * 3;

	@Override
	public void runTest(ClientGameTestContext context) {
		context.getInput().resizeWindow(1280, 720);
		context.runOnClient(Preset.BALANCED::apply);
		context.runOnClient(minecraft -> minecraft.options.renderDistance().set(5));
		context.runOnClient(minecraft -> minecraft.options.chatVisibility().set(ChatVisiblity.HIDDEN));
		context.runOnClient(minecraft -> VoidClient.config.showFps = false);
		try (TestSingleplayerContext world = context.worldBuilder().setUseConsistentSettings(false).create()) {
			world.getServer().runCommand("time set 1000");
			world.getServer().runCommand("weather clear");
			settle(world);
			context.waitTicks(200);
			float[] yaws = {180, 225, 270};
			for (float yaw : yaws) {
				context.getInput().lookAt(yaw, 3);
				context.waitTicks(100);
				context.getInput().holdKey(options -> options.keyUp);
				context.waitTicks(2);
				context.takeScreenshot("promo-hud-" + (int) yaw);
				context.getInput().releaseKey(options -> options.keyUp);
			}
			context.setScreen(() -> new VoidScreen(null));
			context.waitTicks(5);
			context.takeScreenshot("promo-menu");
			context.setScreen(() -> null);
		}
		context.runOnClient(minecraft -> minecraft.options.chatVisibility().set(ChatVisiblity.FULL));
		context.runOnClient(minecraft -> VoidClient.config.showFps = true);
	}

	private static void settle(TestSingleplayerContext world) {
		try {
			world.getConnection().waitForChunksRender(true, RENDER_TIMEOUT);
		} catch (AssertionError e) {
			VoidClient.LOGGER.info("[gametest] chunks still rendering, taking screenshots anyway");
		}
	}
}
