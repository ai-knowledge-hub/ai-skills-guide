// Playwright CLI run-code --filename entry point. Open the review origin first.
async (page) => {
  const origin = new URL(page.url()).origin;
  const routes = [
    ["home", "/"], ["skills", "/skills"], ["agents", "/agents"], ["plugins", "/plugins"], ["tools", "/tools-mcp"],
    ["skill-detail", "/skills/marketing/meta-google-weekly-performance-review"],
    ["agent-detail", "/agents/adtech/ad-platform-control-plane-supervisor"],
    ["plugin-detail", "/plugins/marketing/chatgpt-advertising-experiment-plugin"],
    ["tool-detail", "/tools-mcp/warehouse/bigquery-mcp-query-runner"],
    ["empty", "/skills?q=qa81-no-matching-package"], ["not-found", "/skills/qa81-not-a-real-entry"]
  ];
  await page.context().addInitScript(() => {
    window.__qa81CLS = 0;
    new PerformanceObserver(list => {
      for (const entry of list.getEntries()) if (!entry.hadRecentInput) window.__qa81CLS += entry.value;
    }).observe({type: "layout-shift", buffered: true});
  });
  const rows=[];
  for (const theme of ["dark", "light"]) {
    await page.evaluate(theme => localStorage.setItem("skills-hub-theme",theme),theme);
    for (const width of [320,390,768,1280,1440,1920]) {
      await page.setViewportSize({width,height:width<500?844:1000});
      for (const [name,route] of routes) {
        const response=await page.goto(origin+route,{waitUntil:"networkidle"});
        await page.evaluate(() => document.fonts.ready);
        const result=await page.evaluate(() => {
          const root=document.documentElement;
          const rect=document.querySelector("main h1").getBoundingClientRect();
          const resources=performance.getEntriesByType("resource").filter(r=>r.name.includes("/brand/"));
          return {
            viewport:innerWidth, usable:root.clientWidth, scrollWidth:root.scrollWidth,
            theme:root.dataset.theme, title:document.querySelector("main h1").textContent,
            headingWidth:rect.width, headingInside:rect.left>=0&&rect.right<=root.clientWidth+1,
            cls:window.__qa81CLS,
            artwork:resources.map(r=>({url:new URL(r.name).pathname,bytes:r.encodedBodySize})),
            entranceImage:document.querySelector(".entrance-image")?.currentSrc,
            nativeAnimations:document.getAnimations().length,
            unnamedControls:Array.from(document.querySelectorAll("button,input:not([type=hidden]),select,summary")).filter(el=>el.getClientRects().length&&!(el.getAttribute("aria-label")||el.getAttribute("aria-labelledby")||el.labels?.length||el.textContent?.trim())).map(el=>el.outerHTML.slice(0,140)),
            fonts:getComputedStyle(document.body).fontFamily
          };
        });
        let screenshot;
        if(width===390||width===1440) {
          screenshot=`${name}-${width}-${theme}.png`;
          await page.screenshot({path:screenshot});
        }
        rows.push({name,route,width,requestedTheme:theme,status:response.status(),screenshot,...result});
      }
    }
  }
  return rows;
}
