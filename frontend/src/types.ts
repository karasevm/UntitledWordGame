export interface RoomMember {
  name?: string;
  score: number;
  actionDone: boolean;
}

export interface Answer {
  id: string;
  content: string;
}

export enum GameStage {
  WaitingStage = 0,
  WritingStage,
  VotingStage,
  WinnerStage
}
