const fs = require('fs');
const path = require('path');
const puppeteer = require('puppeteer');
const { spawn, execSync } = require('child_process');

async function main() {
  const rootDir = path.resolve(__dirname, '..');
  const imagesDir = path.join(rootDir, 'docs/images');
  const framesDir = path.join(imagesDir, 'frames');
  fs.mkdirSync(framesDir, { recursive: true });

  console.log('1. Spawning local codekernel studio daemon...');
  const exePath = path.join(rootDir, 'codekernel.exe');
  const server = spawn(exePath, ['studio', '--port=8888'], {
    cwd: rootDir,
    stdio: 'ignore',
  });

  // Wait for server to respond on 8888
  let ready = false;
  for (let i = 0; i < 20; i++) {
    try {
      const res = await fetch('http://127.0.0.1:8888/api/stats');
      if (res.ok) {
        ready = true;
        break;
      }
    } catch (e) {
      await new Promise((r) => setTimeout(r, 300));
    }
  }

  if (!ready) {
    server.kill();
    throw new Error('Server failed to start on port 8888');
  }
  console.log('✓ Studio server is online on http://127.0.0.1:8888');

  try {
    console.log('2. Launching headless browser...');
    const browser = await puppeteer.launch({
      headless: 'new',
      args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,900'],
    });

    const page = await browser.newPage();
    await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });

    let frameIdx = 0;
    const captureFrame = async (count = 1) => {
      for (let c = 0; c < count; c++) {
        const filename = `frame_${String(frameIdx++).padStart(3, '0')}.png`;
        await page.screenshot({ path: path.join(framesDir, filename) });
      }
    };

    // A. Call Graph view
    console.log('3. Capturing Call Graph View...');
    await page.goto('http://127.0.0.1:8888', { waitUntil: 'domcontentloaded' });
    await new Promise((r) => setTimeout(r, 2000));

    await page.screenshot({
      path: path.join(imagesDir, 'screenshot_callgraph.png'),
    });
    await captureFrame(3);

    // B. Symbol Tree View
    console.log('4. Capturing Symbol Tree View...');
    const buttons = await page.$$('header nav button');
    if (buttons.length >= 2) {
      await buttons[1].click(); // Symbol Tree
      await new Promise((r) => setTimeout(r, 1200));

      // Click on symbol in list to show details
      const symBtn = await page.$('.space-y-1 button');
      if (symBtn) {
        await symBtn.click();
        await new Promise((r) => setTimeout(r, 600));
      }

      await page.screenshot({
        path: path.join(imagesDir, 'screenshot_symboltree.png'),
      });
      await captureFrame(3);
    }

    // C. Context Packer Sandbox
    console.log('5. Capturing Context Packer Sandbox...');
    if (buttons.length >= 3) {
      await buttons[2].click(); // Context Packer
      await new Promise((r) => setTimeout(r, 1200));

      await page.screenshot({
        path: path.join(imagesDir, 'screenshot_contextpacker.png'),
      });
      await captureFrame(4);
    }

    // D. File Viewer
    console.log('6. Capturing File Viewer...');
    if (buttons.length >= 4) {
      await buttons[3].click(); // Files
      await new Promise((r) => setTimeout(r, 1200));

      await page.screenshot({
        path: path.join(imagesDir, 'screenshot_fileviewer.png'),
      });
      await captureFrame(3);
    }

    // Return to Call Graph
    if (buttons.length >= 1) {
      await buttons[0].click();
      await new Promise((r) => setTimeout(r, 1000));
      await captureFrame(2);
    }

    await browser.close();
    console.log(`✓ Captured all 4 official screenshots and ${frameIdx} frames.`);

    // Compile animated demo GIF via ffmpeg
    const gifOutput = path.join(imagesDir, 'codekernel_demo.gif');
    console.log('7. Compiling animated demo GIF via ffmpeg...');
    const framePattern = path.join(framesDir, 'frame_%03d.png');

    const ffmpegCmd = `ffmpeg -y -framerate 1.2 -i "${framePattern}" -vf "fps=1.2,scale=1280:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse" "${gifOutput}"`;
    execSync(ffmpegCmd, { stdio: 'inherit' });
    console.log('✓ Successfully generated ' + gifOutput);

    // Clean frames
    fs.rmSync(framesDir, { recursive: true, force: true });
  } finally {
    server.kill();
    console.log('✓ Stopped studio server daemon.');
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
