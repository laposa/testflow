<div class="loading-overlay">
  <svg width="75px" height="75px" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
    <!-- Mask Definition -->
    <mask id="circleMask">
      <!-- Mask Background (completely white fills the entire mask) -->
      <rect width="100" height="100" fill="white" />
      <!-- Moving Dots (black will hide the main circle's stroke) -->
      <circle cx="50" cy="5" r="15" fill="black">
        <animateTransform attributeName="transform" attributeType="XML" type="rotate" from="0 50 50" to="360 50 50" dur="2s" repeatCount="indefinite" />
      </circle>
      <circle cx="50" cy="5" r="15" fill="black">
        <animateTransform attributeName="transform" attributeType="XML" type="rotate" from="30 50 50" to="390 50 50" dur="2s" repeatCount="indefinite" />
      </circle>
      <circle cx="50" cy="5" r="15" fill="black">
        <animateTransform attributeName="transform" attributeType="XML" type="rotate" from="60 50 50" to="420 50 50" dur="2s" repeatCount="indefinite" />
      </circle>
    </mask>

    <!-- Main Circle with Mask Applied -->
    <circle cx="50" cy="50" r="45" stroke="#5f9ea0" fill="none" stroke-width="5" mask="url(#circleMask)" />

    <!-- Dots (Visible) -->
    <circle cx="50" cy="5" r="5" fill="#5f9ea0">
      <animateTransform attributeName="transform" attributeType="XML" type="rotate" from="0 50 50" to="360 50 50" dur="2s" repeatCount="indefinite" />
    </circle>
    <circle cx="50" cy="5" r="5" fill="#5f9ea0">
      <animateTransform attributeName="transform" attributeType="XML" type="rotate" from="30 50 50" to="390 50 50" dur="2s" repeatCount="indefinite" />
    </circle>
    <circle cx="50" cy="5" r="5" fill="#5f9ea0">
      <animateTransform attributeName="transform" attributeType="XML" type="rotate" from="60 50 50" to="420 50 50" dur="2s" repeatCount="indefinite" />
    </circle>

    <path d="M 30,50 L 45,65 L 70,35" stroke="#5f9ea0" stroke-width="5" fill="none" />
  </svg>
</div>