# Attribution

Third-party material in this repository, one row per item, added in the same change as the
material ([conventions](docs/developer/conventions.md#clean-room)).

| Path | Title | Author | Source | Licence |
|---|---|---|---|---|
| `third_party/fyne-terminal/` | fyne-io/terminal at commit `46cd12724ff3`, with Cogpit's patches listed in its `PATCHES.md` | Fyne.io developers | https://github.com/fyne-io/terminal | BSD-3-Clause (`third_party/fyne-terminal/LICENSE`) |
| `AGENTS.md`, `ROADMAP.md`, `docs/developer/*`, `docs/parity/*`, `docs/security/*` | Structure, method and some rule text adapted from PhotoCraft's documentation (`AGENTS.md`, `ROADMAP.md`, `docs/`) at commit `67e3d9be9f2a` | ArtCraft Team and the PhotoCraft contributors | https://github.com/storytold/photocraft | MIT (PhotoCraft is MIT OR Apache-2.0; used under MIT, notice below) |
| `internal/ui/fyneui/theme.go`, `internal/ui/fyneui/termview/termview.go` | The colours of the Solarized Dark and Solarized Light themes (`theme-solarized-dark`, `theme-solarized-light`, VS Code 1.140.0, extension version 10.0.0), copied once into Cogpit's theme tokens and the terminal's sixteen colours, a few adjusted for contrast ([D-22](docs/developer/decisions.md#d-22-colour-theme)); the values are Solarized's palette | VS Code: Microsoft Corporation and contributors. Solarized: Ethan Schoonover | https://github.com/microsoft/vscode (`extensions/theme-solarized-dark`, `extensions/theme-solarized-light`), https://github.com/altercation/solarized | MIT (both, notices below) |
| `internal/helm/bundle/prometheus-29.36.1.tgz` | The prometheus Helm chart 29.36.1 with its subcharts (alertmanager 2.1.0, kube-state-metrics 8.6.0, prometheus-node-exporter 4.59.0, prometheus-pushgateway 3.9.1), embedded unchanged as Cogpit Metrics' bundle ([D-61](docs/developer/decisions.md#d-61-cogpit-metrics-the-bundled-metrics-stack)); SHA-256 `8492d3a0…246217e` as the repository's index lists it | The Prometheus Community chart maintainers | https://github.com/prometheus-community/helm-charts (index https://prometheus-community.github.io/helm-charts) | Apache-2.0 (the chart's `artifacthub.io/license`; Cogpit's `LICENSE` is the same licence's text) |

Facts about Lens Desktop in `docs/parity/` are descriptions of its behaviour with links to its
documentation. No code, text or asset of Lens Desktop, OpenLens or Freelens is used
([D-12](docs/developer/decisions.md#d-12-parity-sources)).

## PhotoCraft licence notice

```text
MIT License

Copyright (c) 2026 ArtCraft Team and the PhotoCraft contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## Solarized and VS Code licence notices

The colours of Solarized Dark and Solarized Light are Ethan Schoonover's palette, as the Solarized
themes of VS Code colour VS Code's interface. Both are MIT licensed; the MIT text is the one
above, with these copyright lines in place of PhotoCraft's:

```text
Copyright (c) 2011 Ethan Schoonover
Copyright (c) Microsoft Corporation. All rights reserved.
```
