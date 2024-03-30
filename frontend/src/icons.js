import { svg } from 'lit';

export default {
  'add': svg`
    <path stroke-linejoin="round" stroke-linecap="round" stroke-miterlimit="10" stroke-width="2" stroke="var(--tp-icon-color)" d="M3 12H21"></path>
    <path stroke-linejoin="round" stroke-linecap="round" stroke-miterlimit="10" stroke-width="2" stroke="var(--tp-icon-color)" d="M12 3V21"></path>
  `,
  'refresh': svg`
    <path fill="var(--tp-icon-color)" d="M17.65,6.35C16.2,4.9 14.21,4 12,4A8,8 0 0,0 4,12A8,8 0 0,0 12,20C15.73,20 18.84,17.45 19.73,14H17.65C16.83,16.33 14.61,18 12,18A6,6 0 0,1 6,12A6,6 0 0,1 12,6C13.66,6 15.14,6.69 16.22,7.78L13,11H20V4L17.65,6.35Z"></path>
  `,
  'delete': svg`
    <path fill="var(--tp-icon-color)" d="M19,4H15.5L14.5,3H9.5L8.5,4H5V6H19M6,19A2,2 0 0,0 8,21H16A2,2 0 0,0 18,19V7H6V19Z" />
  `
};
