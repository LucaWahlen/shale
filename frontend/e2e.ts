
import { chromium } from "playwright";

const b = await chromium.launch();
const page = await b.newPage({ viewport: { width: 1100, height: 900 } });
const errors: string[] = [];
page.on("pageerror", (e) => errors.push("pageerror: " + e.message));
page.on("console", (m) => {
  if (m.type() === "error") errors.push("console: " + m.text().slice(0, 200));
});
page.on("request", (req) => {
  const u = req.url();
  if (u.includes("/api/v1/admin/schedules/") && (req.method() === "POST" || req.method() === "PATCH")) {
    console.log(`### REQ ${req.method()} ${u}`);
    console.log("### BODY:", req.postData());
  }
});
page.on("response", (res) => {
  if (res.status() >= 400 && res.url().includes("/api/")) {
    console.log(`### RESP ${res.status()} ${res.url()}`);
  }
});

await page.goto("http://localhost:8080/admin/login");
await page.fill('input[type="password"]', "admin");
await page.click('button[type="submit"]');
await page.waitForTimeout(900);
console.log("### after login URL:", page.url());
console.log("### main h1:", await page.locator("main h1").first().textContent().catch(() => "(EMPTY MAIN)"));

const sid = await page.evaluate(async () => {
  const res = await fetch("/api/v1/admin/schedules", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title: "E2E Schedule" }),
  });
  return (await res.json()).id;
});
await page.goto("http://localhost:8080/schedules/" + sid);
await page.waitForTimeout(700);

await page.getByRole("button", { name: /add event/i }).click();
await page.waitForTimeout(500);

const seg = (t: string) => page.locator(`[data-slot="date-input-group-segment"][data-type="${t}"]`);
console.log("### segment count day:", await seg("day").count());

await seg("day").click();
await page.keyboard.type("20092026", { delay: 80 });
await page.waitForTimeout(300);
for (const t of ["day", "month", "year"]) {
  console.log(`### ${t} text:`, JSON.stringify(await seg(t).textContent().catch(() => "?")));
}

await seg("hour").click();
await page.keyboard.type("0930", { delay: 80 });
await page.waitForTimeout(300);
for (const t of ["hour", "minute"]) {
  console.log(`### ${t} text:`, JSON.stringify(await seg(t).textContent().catch(() => "?")));
}

await page.fill('input[name="event-name"]', "Typed Event");
await page.screenshot({ path: "/tmp/opencode/e2e-filled.png" });

const saveDisabled = await page.getByRole("button", { name: /^Save$/ }).first().isDisabled();
console.log("### save disabled:", saveDisabled);

await page.getByRole("button", { name: /^Save$/ }).first().click();
await page.waitForTimeout(900);
console.log("### event created:", await page.locator("h3", { hasText: "Typed Event" }).count());
console.log("### toasts:", await page.locator('[data-slot="toast"]').count());
await page.screenshot({ path: "/tmp/opencode/e2e-after-save.png" });

console.log("### ERRORS:", errors.length ? errors.slice(0, 5) : "none");
await b.close();
