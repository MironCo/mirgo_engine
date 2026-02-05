#version 330

in float fragDepth;

out vec4 finalColor;

void main()
{
    // Encode depth (0 to 1) into color buffer
    float depth = fragDepth * 0.5 + 0.5;
    finalColor = vec4(depth, depth, depth, 1.0);
}
