<script lang="ts">
  import type { Answer } from '../types';
  import SkewedButton from './SkewedButton.svelte';
  export let answers: Answer[];
  export let onAnswerVote: (id: string) => void;
  export let disabled: boolean;
</script>

<div>
  {#each answers as answer}
    <SkewedButton
      {disabled}
      onClick={() => onAnswerVote(answer.id)}
      text={answer.content}
    />
  {/each}
</div>

<style lang="scss">
  @import '../colors.scss';
  div {
    display: flex;
    flex-wrap: wrap;
  }
</style>
