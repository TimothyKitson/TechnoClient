package com.voidclient;

import com.mojang.blaze3d.systems.RenderSystem;
import com.mojang.renderpearl.api.device.DeviceInfo;
import com.mojang.renderpearl.api.device.DeviceType;
import com.mojang.renderpearl.api.device.GpuDevice;

import java.lang.management.ManagementFactory;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Locale;

public record Hardware(String gpuName, DeviceType gpuType, long totalRamMb, long maxHeapMb, int cores, boolean nvidiaInstalled) {
	public static Hardware detect() {
		String gpuName = "Unknown GPU";
		DeviceType gpuType = DeviceType.OTHER;
		GpuDevice device = RenderSystem.tryGetDevice();
		if (device != null) {
			DeviceInfo info = device.getDeviceInfo();
			gpuName = info.name();
			gpuType = info.type();
		}
		return new Hardware(gpuName, gpuType, detectTotalRamMb(), Runtime.getRuntime().maxMemory() / (1024 * 1024),
				Runtime.getRuntime().availableProcessors(), detectNvidia());
	}

	private static long detectTotalRamMb() {
		if (ManagementFactory.getOperatingSystemMXBean() instanceof com.sun.management.OperatingSystemMXBean os) {
			return os.getTotalMemorySize() / (1024 * 1024);
		}
		return 0;
	}

	private static boolean detectNvidia() {
		String root = System.getenv("SystemRoot");
		return root != null && Files.exists(Path.of(root, "System32", "nvapi64.dll"));
	}

	public boolean isDiscrete() {
		return gpuType == DeviceType.DISCRETE || (gpuType == DeviceType.OTHER && looksDiscrete());
	}

	private boolean looksDiscrete() {
		String name = gpuName.toLowerCase(Locale.ROOT);
		return name.contains("geforce") || name.contains("rtx") || name.contains("gtx") || name.contains("radeon rx") || name.contains("arc a");
	}

	public boolean onIntegratedWhileNvidiaPresent() {
		return nvidiaInstalled && !isDiscrete() && !gpuName.toLowerCase(Locale.ROOT).contains("nvidia");
	}

	public Preset recommendedPreset() {
		boolean lowRam = totalRamMb < 7 * 1024;
		if (cores <= 2 || lowRam && !isDiscrete()) {
			return Preset.POTATO;
		}
		if (isDiscrete()) {
			return totalRamMb >= 12 * 1024 ? Preset.HIGH : Preset.BALANCED;
		}
		return Preset.BALANCED;
	}

	public long recommendedHeapMb() {
		if (totalRamMb < 7 * 1024) {
			return 2048;
		}
		if (totalRamMb < 12 * 1024) {
			return 3072;
		}
		return 4096;
	}

	public String ramLabel() {
		return Math.round(totalRamMb / 1024.0) + " GB RAM";
	}
}
