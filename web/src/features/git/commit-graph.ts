export type GraphCommit = { id: string; parents: string[] };
export type GraphRow = { lane: number; width: number; edges: { from: number; to: number; parent: string; through: boolean }[] };
// Active lanes carry actual object IDs across rows and page boundaries. Merges
// add lanes, convergence joins an existing lane, roots end their lane.
export function commitGraph(commits: GraphCommit[]): GraphRow[] {
  const lanes: string[] = [];
  return commits.map(commit => {
    let lane = lanes.indexOf(commit.id);
    if (lane < 0) { lane = lanes.length; lanes.push(commit.id); }
    const before = [...lanes];
    lanes.splice(lane, 1);
    let insertion = lane;
    for (const parent of commit.parents) { if (!lanes.includes(parent)) lanes.splice(insertion++, 0, parent); }
    const edges = before.flatMap((id, from) => id === commit.id ? [] : [{ from, to: lanes.indexOf(id), parent: id, through: true }]);
    for (const parent of commit.parents) edges.push({ from: lane, to: lanes.indexOf(parent), parent, through: false });
    return { lane, width: Math.max(before.length, lanes.length, 1), edges };
  });
}
