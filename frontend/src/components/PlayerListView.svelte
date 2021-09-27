<script lang="ts">
  import type { RoomMember } from '../types';
  import { slide } from 'svelte/transition';

  export let players: RoomMember[];
</script>

<div class="container">
  {#each players as player, index (player.name)}
    <div
      class="playerContainer"
      transition:slide|local={{ duration: 100, delay: 100 }}
    >
      <!-- <img
        src={`https://robohash.org/${player.name}.png?set=set4&size=32x32`}
        alt={player.name}
      /> -->
      <img
        src={`https://source.boringavatars.com/beam/32/${player.name}`}
        alt={`player.name's avatar`}
      />
      <span class="text-left">{player.name} {player.actionDone ? '✔' : ''}</span
      >
      <span class="text-right">{player.score}</span>
    </div>
  {/each}
</div>

<style type="text/scss">
  @import '../colors';
  .playerContainer {
    display: flex;
    // max-width: 200px;
    font-size: 24px;
    transform: skewX(10deg);
    padding-left: 6px;
    padding-right: 6px;
    color: $text-color;
    background-color: $element-bg;
    // border-style: solid solid solid solid;
    border-style: none none solid none;
    border-color: #00000070;
    // border-width: 2px;
    // margin-top: 2px;
    // margin-bottom: 2px;
  }
  .text-right {
    font-weight: bold;
    margin-left: auto;
  }
  img {
    padding: 4px;
  }
</style>
