# Turning on Microsoft sign-in (standalone online play)

Standalone mode can run Minecraft without the official launcher. To let people
**sign in with Microsoft** for online play, the launcher needs an **Azure app
client ID**. This is a one-time setup. Until it's set, standalone still works in
**offline mode**, and **Minecraft Launcher mode** already plays online with no
Azure app at all.

A client ID is **not a secret** — it only identifies the app. Each player signs
in with their own Minecraft account, so the client ID is safe to ship in the
launcher.

## 1. Get a directory (one-time)

Microsoft won't let you register an app without an Azure "directory". A brand-new
personal account doesn't have one. To get one for free:

- Sign up for a **free Azure account**: https://azure.microsoft.com/free
  - It asks for a **debit/credit card for identity only — no charge**. A small
    temporary hold may appear and reverses itself. Azure free never auto-bills.
  - The account holder usually must be 18+, so a parent may need to do this.
    That's fine — the client ID is not tied to who creates it.
- This creates a **Default Directory** you can register apps in.

(Students with a school email can instead use https://azure.microsoft.com/free/students — no card.)

## 2. Register the app

1. Go to https://portal.azure.com and make sure you're in your **Default
   Directory** (top-right → Switch directory — not "Microsoft Services").
2. Search **App registrations** → **New registration**.
3. Fill in:
   - **Name:** Void Launcher (users never see this)
   - **Supported account types:** *Accounts in any organizational directory … and
     personal Microsoft accounts (e.g. Skype, Xbox)* — must include personal
     accounts, since Minecraft accounts are personal Microsoft accounts.
   - **Redirect URI:** leave blank (device-code login doesn't use one).
   - **Register**.
4. Left menu → **Authentication** → **Advanced settings → Allow public client
   flows → Yes** → **Save**. (This enables the device-code sign-in the launcher uses.)
5. **Overview** page → copy the **Application (client) ID**.

No client secret and no API permissions are needed — the launcher requests the
`XboxLive.signin` scope at sign-in time.

## 3. Put the client ID in the launcher

Either:

- Set it as the built-in default in `launcher/auth.go`:

  ```go
  const defaultMSAClientID = "YOUR-CLIENT-ID-HERE"
  ```

  then rebuild (`tools/build-site.sh`). Every build of `VoidLauncher.exe` then
  has Microsoft sign-in enabled.

- Or, for a quick local test without rebuilding, set an environment variable
  before launching:

  ```
  VOID_MSA_CLIENT_ID=YOUR-CLIENT-ID-HERE
  ```

  (The env var overrides the built-in default.)

## How the sign-in works (for reference)

The launcher uses the Microsoft **device-code** flow: it shows a short code and a
URL, the player enters the code at that URL in any browser, and the launcher then
exchanges the result through Xbox Live → XSTS → Minecraft services to get a
Minecraft token and profile. See `launcher/auth.go`.
