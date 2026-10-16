"use strict";
const $ = (s) => document.querySelector(s);
const el = (t, c) => { const e = document.createElement(t); if (c) e.className = c; return e; };
const esc = (s) => String(s == null ? "" : s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));

let state = { runMode: "", packs: [], recommendedRamMb: 2048 };
let currentPack = null;
let mcVersions = ["26.3"];
let activeCat = "All";

async function api(path, opts) {
	const r = await fetch(path, opts);
	const text = await r.text();
	let data = {};
	try { data = text ? JSON.parse(text) : {}; } catch { /* non-json */ }
	if (!r.ok || (data && data.error)) throw new Error((data && data.error) || ("Request failed (" + r.status + ")"));
	return data;
}

function coverFor(p) { return (p && p.cover) || window.DEFAULT_COVER; }

/* ---------- Nav ---------- */
function show(view) {
	["packs", "browse", "detail"].forEach((v) => { $("#view-" + v).hidden = v !== view; });
	document.querySelectorAll(".tab").forEach((t) => t.classList.toggle("active", t.dataset.view === view));
}
document.querySelectorAll(".tab").forEach((t) => t.addEventListener("click", () => {
	if (t.dataset.view === "browse") loadBrowse(null);
	show(t.dataset.view);
}));

/* ---------- State / packs ---------- */
async function refresh() {
	state = await api("/api/state");
	state.packs = state.packs || [];
	state.packs.forEach((p) => { p.mods = p.mods || []; });
	$("#runMode").textContent = state.runMode === "standalone" ? "Standalone" : "Minecraft Launcher";
	renderPacks();
	return state;
}

function renderPacks() {
	const list = $("#packList");
	list.innerHTML = "";
	(state.packs || []).forEach((p) => {
		const row = el("div", "pack");
		row.innerHTML =
			`<div class="cover"><img src="${esc(coverFor(p))}" alt=""></div>
			<div class="pinfo">
				<h3>${esc(p.name)}</h3>
				<div class="meta"><span class="chip">MC ${esc(p.mc)}</span><span class="chip loader">${esc(p.loader)}</span></div>
				<div class="count">${p.mods.length} mods · ${Math.round(p.ramMb / 1024 * 10) / 10} GB RAM</div>
			</div>
			<div class="pacts">
				<button class="btn btn-primary btn-small play"><svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg>Play</button>
				<button class="btn btn-ghost btn-small edit">Edit</button>
			</div>`;
		row.querySelector(".play").addEventListener("click", (e) => { e.stopPropagation(); playPack(p); });
		row.querySelector(".edit").addEventListener("click", (e) => { e.stopPropagation(); openPack(p.id); });
		row.addEventListener("click", () => openPack(p.id));
		list.appendChild(row);
	});
	$("#packSub").textContent = (state.packs || []).length + " packs · runs via " + (state.runMode === "standalone" ? "Standalone" : "Minecraft Launcher");
}

/* ---------- Detail ---------- */
function openPack(id) {
	currentPack = (state.packs || []).find((p) => p.id === id);
	if (!currentPack) return;
	const p = currentPack;
	$("#detailName").textContent = p.name;
	$("#detailGlyph").innerHTML = `<img src="${esc(coverFor(p))}" alt="">`;
	$("#detailMc").textContent = "MC " + p.mc;
	$("#detailLoader").textContent = p.loader;
	$("#detailModCount").textContent = p.mods.length;
	$("#detailRam").textContent = (Math.round(p.ramMb / 1024 * 10) / 10) + " GB";
	renderDetailList();
	show("detail");
}
function renderDetailList() {
	const list = $("#detailList");
	list.innerHTML = "";
	if (!currentPack.mods.length) {
		const empty = el("div");
		empty.style.cssText = "color:var(--faint);padding:18px;text-align:center";
		empty.textContent = "No mods yet. Use “Add mods” to browse Modrinth.";
		list.appendChild(empty);
		return;
	}
	currentPack.mods.forEach((m) => {
		const row = el("div", "dmod");
		row.innerHTML =
			`<div class="ic">${m.iconUrl ? `<img src="${esc(m.iconUrl)}" alt="">` : "<span>" + esc((m.name || "?")[0]) + "</span>"}</div>
			<div class="nm"><h3>${esc(m.name)}</h3><div class="by">${esc(m.versionNumber || "")}</div></div>
			<button class="btn btn-ghost btn-small rm">Remove</button>`;
		row.querySelector(".rm").addEventListener("click", async () => {
			try {
				await api("/api/packs/removemod", { method: "POST", body: JSON.stringify({ packId: currentPack.id, projectId: m.projectId }) });
				await refresh();
				currentPack = state.packs.find((x) => x.id === currentPack.id);
				$("#detailModCount").textContent = currentPack.mods.length;
				renderDetailList();
				toast(m.name + " removed");
			} catch (e) { toast(e.message, true); }
		});
		list.appendChild(row);
	});
}
$("#detailBack").addEventListener("click", () => { renderPacks(); show("packs"); });
$("#detailAddMods").addEventListener("click", () => { loadBrowse(currentPack.id); show("browse"); });
$("#detailCover").addEventListener("click", () => $("#detailCoverFile").click());
$("#detailCoverFile").addEventListener("change", (e) => {
	const f = e.target.files[0]; if (!f) return;
	const r = new FileReader();
	r.onload = async () => {
		try {
			await api("/api/packs/cover", { method: "POST", body: JSON.stringify({ id: currentPack.id, cover: r.result }) });
			currentPack.cover = r.result;
			$("#detailGlyph").innerHTML = `<img src="${esc(r.result)}" alt="">`;
			await refresh();
			toast("Cover updated");
		} catch (err) { toast(err.message, true); }
	};
	r.readAsDataURL(f);
});
$("#detailPlay").addEventListener("click", () => playPack(currentPack));

/* ---------- Browse ---------- */
const CATS = ["All", "Performance", "Visual", "Utility", "Social", "Library"];
let browsePackId = null;
let searchTimer = null;

function loadBrowse(packId) {
	browsePackId = packId;
	const pack = packId ? state.packs.find((p) => p.id === packId) : null;
	$("#browseHeading").textContent = pack ? "Add mods to " + pack.name : "Browse Mods";
	$("#browseSub").textContent = "Modrinth · " + (pack ? pack.loader : "Fabric") + " · Minecraft " + (pack ? pack.mc : (mcVersions[0] || "26.3"));
	const pills = $("#pills");
	pills.innerHTML = "";
	CATS.forEach((c) => {
		const b = el("button", "pill" + (c === activeCat ? " on" : ""));
		b.textContent = c;
		b.addEventListener("click", () => { activeCat = c; runSearch(); });
		pills.appendChild(b);
	});
	runSearch();
}

async function runSearch() {
	const grid = $("#browseGrid");
	grid.innerHTML = `<div style="color:var(--faint);padding:24px">Searching Modrinth…</div>`;
	const q = $("#searchInput").value.trim();
	const params = new URLSearchParams();
	if (browsePackId) params.set("packId", browsePackId);
	if (q) params.set("q", q);
	if (activeCat !== "All") params.set("category", activeCat);
	if (!browsePackId) { params.set("mc", mcVersions[0] || "26.3"); params.set("loader", "fabric"); }
	let hits;
	try { hits = await api("/api/search?" + params.toString()); }
	catch (e) { grid.innerHTML = `<div style="color:var(--bad);padding:24px">${esc(e.message)}</div>`; return; }
	grid.innerHTML = "";
	if (!hits.length) { grid.innerHTML = `<div style="color:var(--faint);padding:24px">No mods found.</div>`; return; }
	const pack = browsePackId ? state.packs.find((p) => p.id === browsePackId) : null;
	hits.forEach((h) => {
		const card = el("div", "mod");
		card.innerHTML =
			`<div class="top">
				<div class="ic">${h.icon_url ? `<img src="${esc(h.icon_url)}" alt="">` : "<span>" + esc((h.title || "?")[0]) + "</span>"}</div>
				<div class="nm"><h3>${esc(h.title)}</h3><div class="by">by ${esc(h.author)}</div></div>
			</div>
			<p>${esc(h.description)}</p>
			<div class="tagrow"><span>${esc((h.categories && h.categories[0]) || "mod")}</span><span>⬇ ${formatDownloads(h.downloads)}</span></div>
			<div class="foot">
				<span class="compat">✓ ${pack ? esc(pack.mc + " · " + pack.loader) : "compatible"}</span>
				${h.inPack
					? `<button class="btn btn-small addbtn added"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" style="width:14px;height:14px"><path d="m20 6-11 11-5-5"/></svg>Added</button>`
					: `<button class="btn btn-primary btn-small addbtn">${pack ? "" : ""}<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></svg>Add</button>`}
			</div>`;
		if (!h.inPack) {
			card.querySelector(".addbtn").addEventListener("click", async (ev) => {
				if (!pack) { toast("Open a pack first, then add mods"); return; }
				const btn = ev.currentTarget; btn.disabled = true; btn.textContent = "Adding…";
				try {
					const res = await api("/api/packs/addmod", { method: "POST", body: JSON.stringify({ packId: pack.id, projectId: h.project_id }) });
					await refresh();
					const extra = (res.added || []).length - 1;
					toast(h.title + " added" + (extra > 0 ? " (+" + extra + " dependency" + (extra > 1 ? "ies" : "") + ")" : ""));
					loadBrowse(pack.id);
				} catch (e) { toast(e.message, true); btn.disabled = false; }
			});
		}
		grid.appendChild(card);
	});
}
$("#searchInput").addEventListener("input", () => { clearTimeout(searchTimer); searchTimer = setTimeout(runSearch, 350); });

function formatDownloads(n) {
	if (n >= 1e6) return (n / 1e6).toFixed(1) + "M";
	if (n >= 1e3) return (n / 1e3).toFixed(1) + "K";
	return String(n || 0);
}

/* ---------- Create pack ---------- */
function openNewPack() {
	const scrim = el("div", "scrim");
	const opts = mcVersions.map((v) => `<option>${esc(v)}</option>`).join("");
	scrim.innerHTML =
		`<div class="modal" role="dialog" aria-modal="true">
			<h2>Create a modpack</h2>
			<p class="lead">Pick a Minecraft version and loader. It comes with Void Client and the FPS boost mods already in — remove any you don't want.</p>
			<div class="field"><label for="np-name">Pack name</label><input id="np-name" value="My Awesome Pack"></div>
			<div class="two">
				<div class="field"><label for="np-mc">Minecraft version</label><select id="np-mc">${opts}</select></div>
				<div class="field"><label for="np-loader">Mod loader</label><select id="np-loader"><option>Fabric</option></select></div>
			</div>
			<div class="field"><label>Cover image</label><div class="cover-pick"><div class="cover-prev"><img id="np-prev" src="${esc(window.DEFAULT_COVER)}" alt=""></div><div><button type="button" class="btn btn-ghost btn-small" id="np-pick">Choose image…</button><div id="np-covtxt" style="font-size:12px;color:var(--faint);margin-top:6px">Using the default cover</div></div></div><input type="file" id="np-file" accept="image/*" hidden></div>
			<div class="hint"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M12 8h.01M11 12h1v4h1"/></svg><div>Every mod is checked against this version and loader, and its required dependencies are added for you.</div></div>
			<div class="modal-acts"><button class="btn btn-ghost" data-close>Cancel</button><button class="btn btn-primary" id="np-create">Create pack</button></div>
		</div>`;
	document.body.appendChild(scrim);
	scrim.addEventListener("click", (e) => { if (e.target === scrim) scrim.remove(); });
	scrim.querySelector("[data-close]").addEventListener("click", () => scrim.remove());
	let chosenCover = "";
	scrim.querySelector("#np-pick").addEventListener("click", () => scrim.querySelector("#np-file").click());
	scrim.querySelector("#np-file").addEventListener("change", (e) => {
		const f = e.target.files[0]; if (!f) return;
		const r = new FileReader();
		r.onload = () => { chosenCover = r.result; scrim.querySelector("#np-prev").src = r.result; scrim.querySelector("#np-covtxt").textContent = f.name; };
		r.readAsDataURL(f);
	});
	scrim.querySelector("#np-create").addEventListener("click", async () => {
		const name = scrim.querySelector("#np-name").value.trim() || "Untitled Pack";
		const mc = scrim.querySelector("#np-mc").value;
		const loader = scrim.querySelector("#np-loader").value;
		scrim.remove();
		setBusy('Creating "' + name + '"…\nAdding Void Client and the FPS boost mods.');
		try {
			await api("/api/packs/create", { method: "POST", body: JSON.stringify({ Name: name, MC: mc, Loader: loader, Cover: chosenCover }) });
			clearBusy();
			await refresh();
			show("packs");
			toast('Created "' + name + '" — comes with the FPS boost, remove what you want');
		} catch (e) { clearBusy(); await refresh(); toast(e.message, true); }
	});
}
$("#newPackBtn").addEventListener("click", openNewPack);
$("#newRow").addEventListener("click", openNewPack);

$("#fpsPackBtn").addEventListener("click", async () => {
	setBusy("Building the Void FPS Boost pack…\nAdding Void Client and the speed mods.");
	try {
		await api("/api/packs/fps", { method: "POST", body: "{}" });
		clearBusy();
		await refresh();
		show("packs");
		toast("Void FPS Boost pack created — press Play");
	} catch (e) { clearBusy(); await refresh(); toast(e.message, true); }
});

/* ---------- Install / Play ---------- */
let busy = null;
function setBusy(msg) {
	if (busy) busy.remove();
	busy = el("div", "scrim");
	busy.innerHTML = `<div class="modal" style="text-align:center;max-width:360px"><div class="spinner"></div><div id="busymsg" style="margin-top:14px">${esc(msg)}</div></div>`;
	document.body.appendChild(busy);
}
function clearBusy() { if (busy) { busy.remove(); busy = null; } }

async function playPack(p) {
	if (!p) return;
	setBusy("Setting up " + p.name + "…\nDownloading mods on first run can take a minute.");
	try {
		const res = await api("/api/packs/play", { method: "POST", body: JSON.stringify({ PackID: p.id }) });
		clearBusy();
		if (res.launcherOpened) toast("Opened the Minecraft Launcher — pick " + p.name + " and press Play");
		else toast(p.name + " is ready in the Minecraft Launcher");
	} catch (e) { clearBusy(); toast(e.message, true); }
}

/* ---------- Run-mode chooser ---------- */
function openRunChooser(firstRun) {
	let pick = state.runMode || "minecraft";
	const scrim = el("div", "scrim");
	scrim.innerHTML =
		`<div class="setup" role="dialog" aria-modal="true">
			${firstRun ? '<div class="mark-lg"><img src="' + esc(window.VOID_LOGO) + '" alt="Void"></div>' : ""}
			<h2>${firstRun ? "Welcome to Void Launcher" : "How Void runs Minecraft"}</h2>
			<p class="lead">Choose how Void should run the game. You can change this anytime.</p>
			<div class="choices">
				<button class="choice" data-mode="minecraft">
					<span class="check"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="m20 6-11 11-5-5"/></svg></span>
					<div class="ic2"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="14" rx="2"/><path d="M8 21h8M12 18v3"/></svg></div>
					<h3>Minecraft Launcher <span class="rec">DEFAULT</span></h3>
					<p>Void adds your packs to the official Minecraft Launcher you already use. Nothing extra to install.</p>
				</button>
				<button class="choice" data-mode="standalone">
					<span class="check"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="m20 6-11 11-5-5"/></svg></span>
					<div class="ic2"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 5v14l11-7z"/></svg></div>
					<h3>Standalone <span class="rec" style="color:var(--warn);border-color:var(--warn)">SOON</span></h3>
					<p>Void downloads and runs Minecraft itself, like CurseForge. Coming in a later update.</p>
				</button>
			</div>
			<div class="foot">
				<small>${firstRun ? "You can switch anytime from the top bar." : "Changing this won't affect packs you've already made."}</small>
				<button class="btn btn-primary" id="setup-go">${firstRun ? "Continue" : "Save"}</button>
			</div>
		</div>`;
	document.body.appendChild(scrim);
	const sync = () => scrim.querySelectorAll(".choice").forEach((c) => c.classList.toggle("sel", c.dataset.mode === pick));
	sync();
	scrim.querySelectorAll(".choice").forEach((c) => c.addEventListener("click", () => { pick = c.dataset.mode; sync(); }));
	if (!firstRun) scrim.addEventListener("click", (e) => { if (e.target === scrim) scrim.remove(); });
	scrim.querySelector("#setup-go").addEventListener("click", async () => {
		try {
			await api("/api/runmode", { method: "POST", body: JSON.stringify({ mode: pick }) });
			await refresh();
			scrim.remove();
			toast((firstRun ? "Set up — " : "") + "running via " + (pick === "standalone" ? "Standalone" : "Minecraft Launcher"));
		} catch (e) { toast(e.message, true); }
	});
}
$("#runPill").addEventListener("click", () => openRunChooser(false));

/* ---------- Toast ---------- */
function toast(msg, bad) {
	const t = el("div", "toast");
	const icon = bad
		? `<svg viewBox="0 0 24 24" fill="none" stroke="#f87171" stroke-width="2.4" stroke-linecap="round"><circle cx="12" cy="12" r="9"/><path d="M12 8v5M12 16h.01"/></svg>`
		: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="m20 6-11 11-5-5"/></svg>`;
	t.innerHTML = icon + "<span>" + esc(msg) + "</span>";
	$("#toastWrap").appendChild(t);
	setTimeout(() => { t.style.transition = "opacity .3s, transform .3s"; t.style.opacity = "0"; t.style.transform = "translateY(8px)"; setTimeout(() => t.remove(), 300); }, bad ? 4200 : 2200);
}

/* ---------- Boot ---------- */
(async function boot() {
	try {
		mcVersions = await api("/api/mcversions");
	} catch { mcVersions = ["26.3"]; }
	await refresh();
	if (state.needsSetup) openRunChooser(true);
})();

$("#quitBtn").addEventListener("click", async () => {
	try { await api("/api/quit", { method: "POST", body: "{}" }); } catch {}
	document.body.innerHTML = '<div style="display:grid;place-items:center;height:100vh;color:var(--muted);font-family:var(--body)">Void Launcher closed. You can close this tab.</div>';
});
