import type { Answer, GameStage, RoomMember } from './types';
import hash from 'object-hash';

export class Message {
  author: string;
  chatMessage: string;
  hash: string;
  constructor(author: string, message: string) {
    this.author = author;
    this.chatMessage = message;
    this.hash = hash(Date.now().toString() + author + message);
  }
}

export class Room {
  name?: string | null;
  players: RoomMember[];
  messages: Message[];
  answers?: Answer[];
  gameStage?: GameStage;
  question?: string;
  winner?: RoomMember;
  winnerAnswer?: Answer;
  constructor() {
    this.players = [];
    this.messages = [];
  }

  reset() {
    this.name = null;
    this.players = [];
    this.messages = [];
  }
}
