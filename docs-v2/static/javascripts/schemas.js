const schema_list = [...document.getElementById("schema_list").getElementsByTagName("li")].map(l => l.textContent);
const selectElement = document.getElementById("schema_links")
const majors = {};
const version = getCurrentSchemaVersion()
for (let s of schema_list) {
    let i = 2;
    for (; i < s.length; i++) {
        if (!s[i].match(/^[a-z]+$/)) {
            break;
        }
    }
    let minor = parseInt(s.substr(i));
    let major = s.substr(0, i);

    if (isNaN(minor)) {
        minor = "";
    }
    if (!(major in majors)) {
        majors[major] = [];
    }
    majors[major].push(minor);
}

Object.keys(majors)
    .sort((a, b) => {
        // v3 was released after v3alpha and doc site should show schema in reverse chronological order
        // so we put v3 first.
        if (a === "v3" && b === "v3alpha") {
            return -1
        }
        if (a === "v3alpha" && b === "v3") {
            return 1
        }
        return a > b ? -1 : 1
    })
    .forEach(function(major) {
        majors[major].sort((a,b) => b-a);
        for (const minor of majors[major]) {
            const v = major + minor;
            selectElement.append(new Option(v, v, v === version, v === version));
        }
    });

function selectSchema(selectObject) {
    var value = selectObject.value;
    if (value != "") {
        location.href = `/docs/references/yaml/?version=${value}`
    }
}

function getCurrentSchemaVersion() {
    const versionParam = "?version=";
    const index = window.location.href.indexOf(versionParam);
    if (index === -1) {
        return "";
    } else {
        return window.location.href.substr(index + versionParam.length);
    }
}