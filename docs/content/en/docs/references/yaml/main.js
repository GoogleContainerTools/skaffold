import { html, render } from "https://unpkg.com/lit-html@1.0.0/lit-html.js";
import { unsafeHTML } from "https://unpkg.com/lit-html@1.0.0/directives/unsafe-html.js";

var version;
(async function() {
    let l = new URL(import.meta.url);
    version = l.hash.replace('#skaffold/', '');
    
    const response = await fetch(`/schemas/${version}.json`);
    const json = await response.json();

    render(html`${template(json.definitions, undefined, '#/definitions/SkaffoldPipeline', 0)}`, document.getElementById("table"));
})();

function* template(definitions, parentDefinition, ref, ident) {
  const name = ref.replace('#/definitions/', '');
  
  let allProperties = [];
  let seen = {}
  if (definitions[name].properties) {
      var properties = definitions[name].properties;
      for (var key of definitions[name].preferredOrder) {
          allProperties.push([key, properties[key]]);
          seen[key] = true;
      }
  }
  if (definitions[name].anyOf) {
      for (var anyOf of definitions[name].anyOf) {
          if (anyOf.preferredOrder) {
              for (var key of anyOf.preferredOrder) {
                  if (!seen[key]) {
                      allProperties.push([key, anyOf.properties[key]]);
                      seen[key] = true;
                  }
              }
          }
      }
  }

  let index = -1
  for (var [key, definition] of allProperties) {
    var desc = definition['x-intellij-html-description'];
    let value = definition.default;
    index++;

    if (key === 'apiVersion') {
        value = `skaffold/${version}`
    }
    if (definition.examples) {
        value = definition.examples[0]
    }
    let valueClass = definition.examples ? 'example' : 'value';

    let required = false;
    if (definitions[name].required) {
        for (var requiredName of definitions[name].required) {
            if (requiredName === key) {
                required = true;
                break;
            }
        }
    }
    let keyClass = required ? 'key required' : 'key';

    // Special case for profiles
    if (name === 'Profile') {
        if ((key === 'build') || (key === 'test') || (key === 'deploy')) {
            yield html`
            <tr>
                <td><span class="key" style="margin-left: ${ident * 20}px">${key}:</span> <span class="value">{}</span></td>
                <td><span class="comment">#</span></td>
                <td><span class="comment">${unsafeHTML(desc)}</span></td>
            </tr>
            `;
            continue
        }
    }

    if (definition.$ref) {
        // Check if the referenced description is a final one
        const refName = definition.$ref.replace('#/definitions/', '');
        if (!definitions[refName].properties && !definitions[refName].anyOf) {
            value = '{}'
        }

        yield html`
        <tr>
            <td class="top"><span class="${keyClass}" style="margin-left: ${ident * 20}px">${key}:</span> <span class="${valueClass}">${value}</span></td>
            <td class="top"><span class="comment">#&nbsp;</span></td>
            <td class="top"><span class="comment">${unsafeHTML(desc)}</span></td>
        </tr>
        `;
    } else if (definition.items && definition.items.$ref) {
        yield html`
        <tr>
            <td class="top"><span class="${keyClass}" style="margin-left: ${ident * 20}px">${key}:</span> <span class="${valueClass}">${value}</span></td>
            <td class="top"><span class="comment">#&nbsp;</span></td>
            <td class="top"><span class="comment">${unsafeHTML(desc)}</span></td>
        </tr>
        `;
    } else if (parentDefinition && parentDefinition.type === 'array' && (index == 0)) {
        yield html`
        <tr>
            <td><span class="${keyClass}" style="margin-left: ${(ident - 1) * 20}px">- ${key}:</span> <span class="${valueClass}">${value}</span></td>
            <td><span class="comment">#&nbsp;</span></td>
            <td><span class="comment">${unsafeHTML(desc)}</span></td>
        </tr>
        `;
    } else if ((definition.type === 'array') && value && (value != '[]')) {
        // Parse value to json array
        let values = JSON.parse(value);

        yield html`
        <tr>
            <td><span class="${keyClass}" style="margin-left: ${ident * 20}px">${key}:</span></td>
            <td><span class="comment">#&nbsp;</span></td>
            <td rowspan="${1 + values.length}"><span class="comment">${unsafeHTML(desc)}</span></td>
        </tr>
        `;

        for (var v of values) {
            yield html`
            <tr>
                <td><span class="key" style="margin-left: ${ident * 20}px">- <span class="${valueClass}">${v}</span></span></td>
                <td><span class="comment">#&nbsp;</span></td>
            </tr>
            `;
        }
    } else if ((definition.type === 'object') && value && (value != '{}')) {
        // Parse value to json object
        let values = JSON.parse(value);

        yield html`
        <tr>
            <td><span class="${keyClass}" style="margin-left: ${ident * 20}px">${key}:</span></td>
            <td><span class="comment">#&nbsp;</span></td>
            <td rowspan="${1 + Object.keys(values).length}"><span class="comment">${unsafeHTML(desc)}</span></td>
        </tr>
        `;

        for (var k in values) {
            let v = values[k];

            yield html`
            <tr>
                <td><span class="key" style="margin-left: ${(ident+1) * 20}px"><span class="${valueClass}">${k}: ${v}</span></span></td>
                <td><span class="comment">#&nbsp;</span></td>
            </tr>
            `;
        }
    } else {
        yield html`
        <tr>
            <td><span class="${keyClass}" style="margin-left: ${ident * 20}px">${key}:</span> <span class="${valueClass}">${value}</span></td>
            <td><span class="comment">#&nbsp;</span></td>
            <td><span class="comment">${unsafeHTML(desc)}</span></td>
        </tr>
        `;
    }

    // This definition references another definition
    if (definition.$ref) {
        yield html`
        ${template(definitions, definition, definition.$ref, ident + 1)}
        `;
    }

    // This definition is an array
    if (definition.items && definition.items.$ref) {
        yield html`
            ${template(definitions, definition, definition.items.$ref, ident + 1)}
        `;
    }
  }
}
