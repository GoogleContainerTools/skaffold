import { html, render } from 'https://unpkg.com/lit-html@1.1.2/lit-html.js';
import { unsafeHTML } from 'https://unpkg.com/lit-html@1.1.2/directives/unsafe-html.js';

(async function() {
  const list = document.getElementById('metrics-list');
  const response = await fetch("metrics.json");
  const json = await response.json();

  render(html`
    ${template(json.definitions["skaffoldMeter"])}
  `, list);

  if (location.hash) {
    table.querySelector(location.hash).scrollIntoView();
  }
})();

function* template(struct) {
    for (let p in struct.properties) {
        yield html`<li>${p}<ul><li>${unsafeHTML(struct.properties[p]["x-intellij-html-description"])}</li></ul></li>`
    }
}
