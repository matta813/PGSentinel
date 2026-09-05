import { render } from "@testing-library/react";
import { expect, test } from "vitest";
import favicon from "../../public/favicon.svg?raw";
import html from "../../index.html?raw";
import { BrandMark } from "./BrandMark";

test("mark is decorative beside the product name and inherits theme colour", () => {
  const { container } = render(<BrandMark />);
  const svg = container.querySelector("svg");
  expect(svg).toHaveAttribute("aria-hidden", "true");
  expect(svg).toHaveAttribute("focusable", "false");
  expect(svg).toHaveAttribute("stroke", "currentColor");
  expect(svg).toHaveAttribute("viewBox", "0 0 32 32");
});

test("favicon is accessible and the browser identifies PGSentinel", () => {
  expect(favicon).toContain("<title id=\"title\">PGSentinel</title>");
  expect(html).toContain('<title>PGSentinel</title>');
  expect(html).toContain('rel="icon" type="image/svg+xml" href="/favicon.svg"');
});
