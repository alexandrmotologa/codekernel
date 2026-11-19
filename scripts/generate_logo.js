const fs = require('fs');
const path = require('path');
const { Resvg } = require('@resvg/resvg-js');

function buildLogoSvg() {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="1024" height="1024">
  <defs>
    <clipPath id="squircle-clip">
      <rect x="24" y="24" width="976" height="976" rx="220" />
    </clipPath>

    <linearGradient id="cyan-glow" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#00f5ff"/>
      <stop offset="100%" stop-color="#0284c7"/>
    </linearGradient>

    <linearGradient id="amber-accent" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#fbbf24"/>
      <stop offset="100%" stop-color="#d97706"/>
    </linearGradient>

    <linearGradient id="slate-dark" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" stop-color="#1e293b"/>
      <stop offset="100%" stop-color="#0b0f19"/>
    </linearGradient>

    <filter id="subtle-shadow" x="-10%" y="-10%" width="120%" height="120%">
      <feDropShadow dx="0" dy="16" stdDeviation="20" flood-color="#000000" flood-opacity="0.14" />
    </filter>
  </defs>

  <!-- Luxury White Squircle Container -->
  <rect x="24" y="24" width="976" height="976" rx="220" fill="#ffffff" stroke="#e2e8f0" stroke-width="6" />

  <g clip-path="url(#squircle-clip)">
    <g transform="translate(512, 512)" filter="url(#subtle-shadow)">

      <!-- Hexagonal Architectural Gateway -->
      <polygon points="
        0,-390
        338,-195
        338,195
        0,390
        -338,195
        -338,-195
      " fill="none" stroke="#0f172a" stroke-width="36" stroke-linejoin="round" />

      <polygon points="
        0,-355
        307,-177
        307,177
        0,355
        -307,177
        -307,-177
      " fill="none" stroke="#00f5ff" stroke-width="4" opacity="0.4" stroke-dasharray="16, 12" />

      <!-- Watertight Base Silhouette: Symmetrical Origami Owl (Noctua) -->
      <path d="
        M 0 -280
        L 90 -295 L 170 -260 L 220 -180 L 210 -40 L 230 80 L 190 230 L 110 320 L 0 345
        L -110 320 L -190 230 L -230 80 L -210 -40 L -220 -180 L -170 -260 L -90 -295
        Z
      " fill="#0b0f19" />

      <!-- Symmetrical Origami Facets: Forehead & Ear Tufts -->
      <!-- Left Ear Tuft -->
      <polygon points="0,-280 -90,-295 -140,-250 -60,-220" fill="#1e293b" />
      <polygon points="-90,-295 -170,-260 -140,-250" fill="#0f172a" />
      <!-- Right Ear Tuft -->
      <polygon points="0,-280 90,-295 140,-250 60,-220" fill="#334155" />
      <polygon points="90,-295 170,-260 140,-250" fill="#1e293b" />

      <!-- Central Brow & Crown -->
      <polygon points="0,-280 -60,-220 0,-170" fill="#475569" />
      <polygon points="0,-280 60,-220 0,-170" fill="#64748b" />

      <!-- Lateral Temples -->
      <polygon points="-140,-250 -60,-220 -130,-140 -200,-170" fill="#1e293b" />
      <polygon points="140,-250 60,-220 130,-140 200,-170" fill="#334155" />

      <!-- Outer Cheeks & Heart-Shaped Facial Disc -->
      <polygon points="-200,-170 -130,-140 -150,-40 -210,-40" fill="#0f172a" />
      <polygon points="200,-170 130,-140 150,-40 210,-40" fill="#1e293b" />

      <polygon points="-130,-140 0,-170 -50,-80" fill="#334155" />
      <polygon points="130,-140 0,-170 50,-80" fill="#475569" />

      <polygon points="-130,-140 -50,-80 -120,-30 -150,-40" fill="#1e293b" />
      <polygon points="130,-140 50,-80 120,-30 150,-40" fill="#334155" />

      <!-- Geometric Almond Predatory Optics (Eyes) -->
      <!-- Left Eye Orbit & Sclera -->
      <polygon points="-110,-100 -45,-85 -70,-55 -125,-70" fill="#0b0f19" />
      <polygon points="-105,-95 -50,-83 -72,-60 -120,-72" fill="url(#cyan-glow)" />
      <!-- Left Pupil slit -->
      <polygon points="-88,-92 -76,-66 -84,-62 -94,-88" fill="#070a10" />

      <!-- Right Eye Orbit & Sclera -->
      <polygon points="110,-100 45,-85 70,-55 125,-70" fill="#0b0f19" />
      <polygon points="105,-95 50,-83 72,-60 120,-72" fill="url(#cyan-glow)" />
      <!-- Right Pupil slit -->
      <polygon points="88,-92 76,-66 84,-62 94,-88" fill="#070a10" />

      <!-- Precision Geometric Beak -->
      <polygon points="0,-100 -25,-40 0,10" fill="#d97706" />
      <polygon points="0,-100 25,-40 0,10" fill="url(#amber-accent)" />
      <polygon points="0,10 -20,60 0,75" fill="#b45309" />
      <polygon points="0,10 20,60 0,75" fill="#d97706" />

      <!-- Lower Muzzle & Throat (Symmetrical Slate-Silver) -->
      <polygon points="-50,-80 -25,-40 0,10 -60,40 -120,-30" fill="#64748b" />
      <polygon points="50,-80 25,-40 0,10 60,40 120,-30" fill="#94a3b8" />

      <polygon points="-60,40 0,75 0,140 -70,120" fill="#334155" />
      <polygon points="60,40 0,75 0,140 70,120" fill="#475569" />

      <!-- Obsidian Breastplate & Folded Wing Contours -->
      <polygon points="-150,-40 -60,40 -70,120 -160,80 -230,80" fill="#0f172a" />
      <polygon points="150,-40 60,40 70,120 160,80 230,80" fill="#1e293b" />

      <polygon points="-70,120 0,140 0,260 -90,240 -160,190" fill="#1e293b" />
      <polygon points="70,120 0,140 0,260 90,240 160,190" fill="#334155" />

      <polygon points="-160,80 -70,120 -160,190 -210,170" fill="#0b0f19" />
      <polygon points="160,80 70,120 160,190 210,170" fill="#1e293b" />

      <!-- Lower Tail Feathers & Baseline Keel -->
      <polygon points="0,140 0,330 -50,290 -90,240" fill="#0f172a" />
      <polygon points="0,140 0,330 50,290 90,240" fill="#1e293b" />
      <polygon points="0,260 0,345 50,290" fill="#334155" />
      <polygon points="0,260 0,345 -50,290" fill="#1e293b" />

      <!-- Cyan Node Accents on Hex Corners -->
      <circle cx="0" cy="-390" r="8" fill="#00f5ff" />
      <circle cx="338" cy="-195" r="8" fill="#00f5ff" />
      <circle cx="338" cy="195" r="8" fill="#00f5ff" />
      <circle cx="0" cy="390" r="8" fill="#00f5ff" />
      <circle cx="-338" cy="195" r="8" fill="#00f5ff" />
      <circle cx="-338" cy="-195" r="8" fill="#00f5ff" />

    </g>
  </g>
</svg>`;
}

async function main() {
  const outputDir = path.resolve(__dirname, '../docs/images');
  fs.mkdirSync(outputDir, { recursive: true });

  const svg = buildLogoSvg();
  const svgPath = path.join(outputDir, 'logo.svg');
  const pngPath = path.join(outputDir, 'logo.png');

  fs.writeFileSync(svgPath, svg, 'utf8');
  console.log('✓ Wrote ' + svgPath);

  const resvg = new Resvg(svg, {
    fitTo: { mode: 'width', value: 1024 }
  });
  const pngData = resvg.render().asPng();
  fs.writeFileSync(pngPath, pngData);
  console.log('✓ Rendered ' + pngPath + ' (1024x1024)');
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
