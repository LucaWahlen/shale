// Compare: plain RAC DateField vs the app's ShaleDatePicker composition.
import { GlobalRegistrator } from "@happy-dom/global-registrator";
import React from "react";
import { createRoot } from "react-dom/client";
import { DateField as RACDateField, DateInput as RACDateInput, DateSegment as RACDateSegment, I18nProvider as RACI18n } from "react-aria-components";

GlobalRegistrator.register({ url: "http://localhost:8080/" });
document.body.innerHTML = '<div id="root"></div>';

const fakeSel: any = {
  rangeCount: 0, isCollapsed: true, anchorNode: null, focusNode: null,
  collapse() {}, collapseToEnd() {}, collapseToStart() {}, setBaseAndExtent() {},
  selectAllChildren() {}, extend() {}, addRange() {}, removeAllRanges() {},
  containsNode: () => false, toString: () => "", getRangeAt: () => { throw new Error(); },
};
(document as any).getSelection = () => fakeSel;
(window as any).getSelection = () => fakeSel;

const { initI18n } = await import("./src/i18n");
initI18n("en");
const { ShaleDatePicker } = await import("./src/components/DateTimeFields");

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const type = (el: Element, keys: string[]) => {
  for (const k of keys) el.dispatchEvent(new KeyboardEvent("keydown", { key: k, bubbles: true, cancelable: true }));
};
const segs = () => Array.from(document.querySelectorAll('[data-slot="date-input-group-segment"], .react-aria-DateInput [data-placeholder], [role="spinbutton"]')) as HTMLElement[];

async function probe(name: string, el: React.ReactNode) {
  const d = document.createElement("div");
  document.body.appendChild(d);
  const r = createRoot(d);
  r.render(<RACI18n locale="en-GB">{el}</RACI18n>);
  await sleep(300);
  const out: { date?: string } = {};
  // re-query segments scoped to this host
  const list = Array.from(d.querySelectorAll('[role="spinbutton"]')).filter(
    (s) => (s as HTMLElement).getAttribute("data-type") !== "literal",
  ) as HTMLElement[];
  const day = list[0];
  day?.focus();
  type(day, ["2", "0"]);
  await sleep(120);
  const month = list[1];
  month?.focus();
  type(month, ["9"]);
  const year = list[2];
  year?.focus();
  type(year, ["2", "0", "2", "6"]);
  await sleep(200);
  console.log(`### ${name}: segments=${list.map((s) => s.getAttribute("data-type")).join("/")} dayText=${JSON.stringify(day?.textContent)} valuetext=${JSON.stringify(day?.getAttribute("aria-valuetext"))}`);
  r.unmount();
  d.remove();
}

let racOut = "";
await probe(
  "plain RAC",
  <RACDateField onChange={(v) => { racOut = v.toString(); console.log("RAC onChange:", v.toString()); }}>
    <RACDateInput>{(s) => <RACDateSegment segment={s} />}</RACDateInput>
  </RACDateField>,
);
console.log("RAC final value:", JSON.stringify(racOut));

let shaleOut = "";
await probe(
  "ShaleDatePicker",
  <ShaleDatePicker label="Date" value="" onChange={(v: string) => { shaleOut = v; console.log("shale onChange:", v); }} />,
);
console.log("shale final value:", JSON.stringify(shaleOut));
process.exit(0);
