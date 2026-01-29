let treeData = {
    name: "root",
    children: []
};

let selectedNode = null;

const svg = d3.select("#tree").append("svg")
    .attr("width", 800)
    .attr("height", 400);

const g = svg.append("g").attr("transform", "translate(40, 0)");

const treeLayout = d3.tree().size([360, 300]);

function updateTree() {
    const root = d3.hierarchy(treeData);
    treeLayout(root);

    const links = g.selectAll(".link")
        .data(root.links())
        .enter().append("path")
        .attr("class", "link")
        .attr("d", d3.linkRadial()
            .angle(d => d.x * Math.PI / 180)
            .radius(d => d.y));

    const nodes = g.selectAll(".node")
        .data(root.descendants())
        .enter().append("g")
        .attr("class", "node")
        .attr("transform", d => `translate(${d.x}, ${d.y})`)
        .on("click", (event, d) => {
            selectedNode = d;
            document.getElementById("cmdName").value = d.data.name;
            document.getElementById("cmdUse").value = d.data.use || "";
            document.getElementById("cmdShort").value = d.data.short || "";
        });

    nodes.append("circle").attr("r", 10);
    nodes.append("text")
        .attr("dy", "0.31em")
        .attr("x", d => d.x < 180 ? 15 : -15)
        .style("text-anchor", d => d.x < 180 ? "start" : "end")
        .text(d => d.data.name);
}

updateTree();

function addCommand() {
    const name = document.getElementById("cmdName").value;
    const use = document.getElementById("cmdUse").value;
    const short = document.getElementById("cmdShort").value;

    if (!selectedNode) {
        alert("Select a parent node first");
        return;
    }

    if (!selectedNode.data.children) {
        selectedNode.data.children = [];
    }

    selectedNode.data.children.push({
        name: name,
        use: use,
        short: short,
        children: []
    });

    updateTree();
}

function addFlag() {
    document.getElementById("flagForm").style.display = "block";
}

function saveFlag() {
    const flagName = document.getElementById("flagName").value;
    const flagType = document.getElementById("flagType").value;
    const flagDesc = document.getElementById("flagDesc").value;

    if (!selectedNode) {
        alert("Select a command node first");
        return;
    }

    if (!selectedNode.data.flags) {
        selectedNode.data.flags = [];
    }

    selectedNode.data.flags.push({
        name: flagName,
        type: flagType,
        description: flagDesc
    });

    document.getElementById("flagForm").style.display = "none";
    updateTree();
}

function generateCode() {
    fetch('/generate', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(treeData),
    })
    .then(response => response.text())
    .then(code => {
        document.getElementById("codeOutput").textContent = code;
    });
}