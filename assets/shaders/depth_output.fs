#version 330

// Fragment shader that outputs depth as color
// Allows depth buffer to be sampled as a regular color texture

in float fragDepth;

out vec4 finalColor;

void main() {
    // Output depth as grayscale - using gl_FragCoord.z is more accurate
    float depth = gl_FragCoord.z;
    finalColor = vec4(depth, depth, depth, 1.0);
}
