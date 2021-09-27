<script lang="typescript">
  import { scale } from 'svelte/transition';
  import { GameStage } from './types';
  import type { RoomMember } from './types';
  import AnswerView from './components/AnswerView.svelte';
  import PlayerListView from './components/PlayerListView.svelte';
  import SkewedButton from './components/SkewedButton.svelte';
  import { Message, Room } from './classes';
  import SkewedInput from './components/SkewedInput.svelte';
  import WritingView from './components/WritingView.svelte';
  import Toast from './components/Toast.svelte';

  let inputValue = 'a';
  let outputLog: string[] = [];
  let params = new URL(document.location.toString()).searchParams;

  // Service
  let ws: WebSocket;
  let backoffTimeout = 10;
  let jwt = sessionStorage.getItem('fgame_jwt');

  // State
  let player: RoomMember = {
    actionDone: false,
    score: 0
  };
  let room = new Room();
  let playerCount: number;
  let toastMessage = '';
  let toastTimeout: number;

  // Input fields
  let joinRoomName = '';

  /**
   * Display toast message
   * @param {string} message - message text
   * @param {number} [timeout=5] - toast timeout
   */
  const toaster = (message: string, timeout = 5) => {
    clearTimeout(toastTimeout);
    toastMessage = message;
    toastTimeout = setTimeout(() => {
      toastMessage = '';
    }, timeout * 1000);
  };

  /**
   * Handle server errors
   * @param {number} errorCode - numeric error id
   * @param {string} errorText - fallback error text
   */
  const wsErrorHandler = (errorCode: number, errorText: string) => {
    switch (errorCode) {
      case 10:
        sessionStorage.removeItem('fgame_jwt');
        break;
      case 11:
        toaster('Name already taken');
        break;
      case 20:
        toaster('Room not found');
        break;
      case 25:
        toaster('Room is full');
        break;
      default:
        if (errorText.length != 0) {
          toaster(errorText);
        }
        break;
    }
  };
  /**
   * Handle incoming websocket messages
   * @param data websocket message content
   * @todo Add type guards
   */
  const wsHandler = (data: any) => {
    console.log(data);
    switch (data.msgType) {
      case 'jwt':
        sessionStorage.setItem('fgame_jwt', data.data);
        break;
      case 'self':
        player.name = data.name;
        room.name = data.room;
        player.actionDone = data.actionDone;
        console.log(JSON.stringify(params));
        if (params.has('j')) {
          const tmpParam = params.get('j');
          if (tmpParam == null) {
            break;
          }
          // leave if player is already in a room
          if (room.name != null && room.name.length === 8) {
            leaveRoomHandler();
            break;
          }
          joinRoomName = tmpParam;
          params.delete('j');
          history.replaceState(null, '', '/');
          joinRoomHandler();
        }
        break;
      case 'roomState':
        console.log(`Got new state:`, data);
        room.players = data.players;
        room.answers = data.answers;
        room.gameStage = data.gameStage;
        room.winner = data.winner;
        room.winnerAnswer = data.winnerAnswer;
        room.question = data.question;
        joinRoomName = '';
        console.log(`New state:`, room);
        break;
      case 'status':
        playerCount = data.playerCount;
        break;
      case 'chat':
        room.messages.unshift(new Message(data.author, data.chatMessage));
        break;
      case 'error':
        if (
          typeof data.errorCode === 'number' &&
          typeof data.error === 'string'
        ) {
          wsErrorHandler(data.errorCode, data.error);
        }
        break;
      default:
        break;
    }
    room = room; // svelte notify state update
  };

  const logger = (message: string) => {
    outputLog = [
      ...outputLog,
      `${new Date().toLocaleTimeString('en-US', {
        hour12: false,
        hour: 'numeric',
        minute: 'numeric',
        second: 'numeric'
      })}: ${message}`
    ];
  };

  const wsConnect = () => {
    if (ws !== undefined) {
      return;
    }
    ws = new WebSocket(
      `ws${window.location.hostname === 'localhost' ? '' : 's'}://${
        window.location.hostname
      }:${
        window.location.hostname === 'localhost' ? '8080' : window.location.port
      }/ws`
    );
    ws.onopen = () => {
      ws = ws;
      logger('OPEN');
      if (typeof jwt === 'string') {
        ws.send(JSON.stringify({ action: 'login', data: jwt }));
      }
    };
    ws.onclose = () => {
      ws = ws;
    };
    ws.onmessage = (evt: MessageEvent) => {
      ws = ws;
      logger(`MESSAGE: ${evt.data}`);
      // outputLog = [...outputLog, 'RESPONSE: ' + evt.data];
      wsHandler(JSON.parse(evt.data));
    };
    ws.onerror = function (this: WebSocket, evt: any) {
      ws = ws;
      logger(`ERROR: ${evt.data}`);
      // outputLog = [...outputLog, 'ERROR: ' + evt.data];
      setTimeout(() => {
        wsConnect();
        backoffTimeout *= 2;
      }, backoffTimeout * 1000);
      return null;
    };
  };
  const registerHandler = () => {
    ws.send(JSON.stringify({ action: 'register', data: inputValue }));
  };

  const createRoomHandler = () => {
    ws.send(JSON.stringify({ action: 'createRoom', data: '' }));
  };

  const joinRoomHandler = () => {
    ws.send(
      JSON.stringify({ action: 'joinRoom', data: joinRoomName.toUpperCase() })
    );
  };

  const leaveRoomHandler = () => {
    ws.send(JSON.stringify({ action: 'leaveRoom' }));
    room.reset();
  };

  const chatHandler = (chatMessage: string) => {
    ws.send(JSON.stringify({ action: 'sendMessage', data: chatMessage }));
  };

  const voteHandler = (id: string) => {
    ws.send(JSON.stringify({ action: 'voteAnswer', data: id }));
  };

  const startHandler = () => {
    ws.send(JSON.stringify({ action: 'startGame' }));
  };

  const answerSubmitHandler = (answer: string) => {
    ws.send(JSON.stringify({ action: 'sendAnswer', data: answer }));
  };

  wsConnect();
</script>

<div class="container">
  <div class="header">
    <h1>Untitled Word Game</h1>
    {#if typeof playerCount !== 'undefined'}
      <h4>There are {playerCount} players online</h4>
    {/if}
    <h3>
      You are {player.name == null ? 'unregistered' : player.name}
      {#if room.name != null && room.name.length > 0}
        in room {room.name}
      {/if}
    </h3>
  </div>
  <main>
    <div class="left">
      {#if player.name == null}
        <form
          class="registerForm"
          on:submit|preventDefault={registerHandler}
          in:scale={{ duration: 100 }}
        >
          <SkewedInput bind:value={inputValue} />
          <SkewedButton
            disabled={typeof ws === 'undefined' || ws.readyState != 1}
            text="Register"
            submit
          />
        </form>
      {/if}
      {#if player.name != null}
        {#if room.name != null && room.name.length > 0}
          <div class="metaContainer" in:scale={{ duration: 100, delay: 100 }}>
            <div class="controlsContainer">
              <SkewedButton onClick={leaveRoomHandler} text="Leave room" />
              <SkewedButton
                text="Copy invite link"
                onClick={() =>
                  navigator.clipboard
                    .writeText(
                      `${location.protocol}//${location.host}${location.pathname}?j=${room.name}`
                    )
                    .then(() => toaster('Copied successfully'))}
              />
              {#if room.gameStage === GameStage.WaitingStage && room.players[0]?.name === player.name && room.players.length > 1}
                <SkewedButton onClick={startHandler} text="Start game" />
              {/if}
            </div>
            <div>
              <PlayerListView players={room.players} />
            </div>
            <!-- <div>
              <Chat {chatHandler} {room} />
            </div> -->
          </div>
        {:else}
          <div
            class="createJoinContainer"
            in:scale={{ duration: 100, delay: 50 }}
          >
            <SkewedButton onClick={createRoomHandler} text="Create new room" />
            <h2>OR</h2>
            <form autocomplete="off" on:submit|preventDefault={joinRoomHandler}>
              <SkewedInput bind:value={joinRoomName} />
              <SkewedButton
                disabled={joinRoomName.length !== 8}
                text="Join room"
                submit
              />
            </form>
          </div>
        {/if}

        <!-- <Debugger {ws} /> -->
        {#if room.name != null && room.name.length > 0}
          {#if room.gameStage === GameStage.WritingStage && !player.actionDone && room.question != null}
            <WritingView
              question={room.question}
              onAnswerSubmit={answerSubmitHandler}
            />
          {:else if room.gameStage === GameStage.VotingStage && room.answers != null}
            <div in:scale={{ duration: 100 }}>
              <h3>{room.question}</h3>
              <AnswerView
                answers={room.answers}
                onAnswerVote={voteHandler}
                disabled={player.actionDone}
              />
            </div>
          {:else if room.gameStage === GameStage.WinnerStage && room.winner != null && room.winnerAnswer != null}
            <div class="winnerView" in:scale={{ duration: 100 }}>
              <h1>
                {room.winner.name} has won the round with the answer {room
                  .winnerAnswer.content}
              </h1>
            </div>
          {/if}
        {/if}
      {/if}
    </div>
    <!-- <div class="right">
    {#if outputLog !== undefined}
      {#each outputLog.slice(-10) as line, index}
        <div class={index % 2 === 1 ? 'dark' : ''}>{line}</div>
      {/each}
    {/if}
  </div> -->
  </main>
</div>
<Toast text={toastMessage} />

<style type="text/scss">
  @import 'colors';
  .container {
    display: grid;
    justify-items: center;
    grid-template-rows: [row1-start] 200px;
    grid-template-columns: 5% [main] 90% 5%;
    width: 100%;
    height: 100%;
  }
  .header,
  main {
    grid-column: main;
  }
  .header {
    text-align: center;
  }
  .createJoinContainer {
    display: flex;
    flex-direction: row;
  }
  @media (max-width: 850px) {
    .createJoinContainer {
      flex-direction: column;
      text-align: center;
    }
    .createJoinContainer form {
      display: inline-grid;
    }
  }
  .controlsContainer {
    display: flex;
    flex-wrap: wrap;
  }
  .metaContainer {
    max-width: 410px;
    margin: auto;
  }
  .registerForm {
    display: flex;
    flex-wrap: wrap;
  }
  .winnerView {
    word-break: break-word;
    text-align: center;
    @media (max-width: 450px) {
      font-size: 4vw;
    }
  }
</style>
