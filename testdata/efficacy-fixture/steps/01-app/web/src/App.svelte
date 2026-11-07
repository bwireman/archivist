<script>
  let notes = [];
  let title = "";
  let body = "";

  async function load() {
    const res = await fetch("/api/notes?notebook_id=nb-inbox");
    notes = await res.json();
  }

  async function createNote() {
    await fetch("/api/notes", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ notebook_id: "nb-inbox", title, body }),
    });
    title = "";
    body = "";
    await load();
  }

  async function remove(id) {
    await fetch("/api/notes/" + id, { method: "DELETE" });
    await load();
  }
</script>

<main>
  <h1>Notes</h1>
  <form onsubmit={(e) => { e.preventDefault(); createNote(); }}>
    <input bind:value={title} placeholder="Title" />
    <input bind:value={body} placeholder="Body" />
    <button type="submit">Add</button>
  </form>
  <ul>
    {#each notes as item}
      <li>
        {item.title}
        <button type="button" onclick={() => remove(item.id)}>Delete</button>
      </li>
    {/each}
  </ul>
</main>
