<script lang="ts">
  import { scale } from 'svelte/transition';

  import SkewedButton from './SkewedButton.svelte';
  import SkewedInput from './SkewedInput.svelte';
  export let onAnswerSubmit: (id: string) => void;
  export let question: string;

  let answerFieldValue = '';

  const answerSubmitHandler = () => {
    if (answerFieldValue.length > 0) {
      onAnswerSubmit(answerFieldValue);
    }
  };
</script>

<div in:scale={{ duration: 100 }}>
  <h3>{question}</h3>
  <form
    class="container"
    autocomplete="off"
    on:submit|preventDefault={answerSubmitHandler}
  >
    <!-- <label for="answerField">Enter your answer</label> -->
    <SkewedInput bind:value={answerFieldValue} />
    <SkewedButton submit text="Submit" />
  </form>
</div>

<style lang="scss">
  h3 {
    color: black;
  }
  .container {
    max-width: 410px;
    margin-left: auto;
    margin-right: auto;
    display: flex;
    flex-direction: row;
    flex-wrap: wrap;
  }
</style>
