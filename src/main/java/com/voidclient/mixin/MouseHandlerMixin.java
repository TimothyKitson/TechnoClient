package com.voidclient.mixin;

import com.mojang.blaze3d.platform.InputConstants;
import com.voidclient.VoidClient;
import com.voidclient.Zoom;
import net.minecraft.client.Minecraft;
import net.minecraft.client.MouseHandler;
import net.minecraft.client.input.MouseButtonInfo;
import org.spongepowered.asm.mixin.Final;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.Shadow;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfo;

@Mixin(MouseHandler.class)
public class MouseHandlerMixin {
	@Shadow
	@Final
	private Minecraft minecraft;

	@Shadow
	private double accumulatedDX;

	@Shadow
	private double accumulatedDY;

	@Inject(method = "onButton", at = @At("HEAD"))
	private void voidclient$countClick(long window, MouseButtonInfo info, int action, CallbackInfo ci) {
		if (action == InputConstants.PRESS && window == minecraft.getWindow().handle() && minecraft.gui.screen() == null) {
			VoidClient.onClick(info.button());
		}
	}

	@Inject(method = "onScroll", at = @At("HEAD"), cancellable = true)
	private void voidclient$zoomScroll(long window, double xOffset, double yOffset, CallbackInfo ci) {
		if (Zoom.isActive() && yOffset != 0 && window == minecraft.getWindow().handle() && minecraft.gui.screen() == null) {
			Zoom.scroll(Math.signum(yOffset));
			ci.cancel();
		}
	}

	@Inject(method = "turnPlayer", at = @At("HEAD"))
	private void voidclient$zoomSensitivity(double movementTime, CallbackInfo ci) {
		double scale = Zoom.sensitivityScale();
		if (scale != 1.0) {
			accumulatedDX *= scale;
			accumulatedDY *= scale;
		}
	}
}
