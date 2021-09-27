<script lang="typescript">
  export let ws: WebSocket;
  let action = '';
  let data = '';
  const handler = () => {
    ws.send(JSON.stringify({ action, data }));
  };
</script>

<form on:submit|preventDefault={handler}>
  <label for="action">Action</label>
  <input name="action" type="text" bind:value={action} />
  <label for="data">Data</label>
  <input name="data" type="text" bind:value={data} />
  <input type="submit" value="Submit" />
</form>
