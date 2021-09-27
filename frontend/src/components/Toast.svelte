<script lang="typescript">
  import { fade } from 'svelte/transition';
  import { quartIn } from 'svelte/easing';
  export let text: string;
</script>

{#if text.length > 0}
  <div class="notifications">
    <div class="toast" transition:fade={{ duration: 300 }}>
      <div class="content">{text}</div>
    </div>
  </div>
{/if}

<style>
  .notifications {
    position: fixed;
    left: 0;
    margin: 0 auto;
    padding: 0;
    z-index: 9999;
    display: flex;
    flex-direction: column;
    justify-content: flex-start;
    align-items: center;
    pointer-events: none;
  }
  @media (max-width: 850px) {
    .notifications {
      bottom: 10%;
      right: 0;
    }
  }
  @media (min-width: 850px) {
    .notifications {
      top: 50px;
      right: 50px;
    }
    .toast {
      align-self: flex-end;
    }
  }

  .toast {
    flex: 0 0 auto;
    margin-bottom: 10px;
    background-color: rgba(50, 50, 50, 0.959);
    border-radius: 25px;
  }

  .content {
    padding: 15px;
    color: white;
    font-weight: 500;
  }
</style>
