package com.voidclient.mixin;

import com.voidclient.Zoom;
import net.minecraft.client.Camera;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfoReturnable;

@Mixin(Camera.class)
public class CameraMixin {
	@Inject(method = "calculateFov", at = @At("RETURN"), cancellable = true)
	private void voidclient$zoom(float partialTick, CallbackInfoReturnable<Float> cir) {
		double divisor = Zoom.divisor();
		if (divisor != 1.0) {
			cir.setReturnValue((float) (cir.getReturnValueF() / divisor));
		}
	}
}
