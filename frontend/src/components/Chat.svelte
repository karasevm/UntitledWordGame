<script lang="ts">
  import type { Room } from '../classes';
  import { slide } from 'svelte/transition';
  import SkewedButton from './SkewedButton.svelte';
  import SkewedInput from './SkewedInput.svelte';
  export let chatHandler: (message: string) => void;
  export let room: Room;
  let chatMessage = '';

  const handleSend = () => {
    if (chatMessage.length === 0) return;
    chatHandler(chatMessage);
    chatMessage = '';
  };
</script>

<h2>chat</h2>
<div>
  <form on:submit|preventDefault={handleSend}>
    <SkewedInput bind:value={chatMessage} />
    <SkewedButton submit text="send" />
  </form>
</div>
<div class="chatContainer">
  {#each room.messages as message, index (message.hash)}
    <div transition:slide|local={{ duration: 100 }}>
      {message.author}:{message.chatMessage}
    </div>
  {/each}
</div>

<style lang="scss">
  @import '../colors.scss';
  .chatContainer {
    transition: height 0.5s cubic-bezier(0.16, 1, 0.3, 1);
    overflow: auto;
    max-height: 7em;
  }
</style>
