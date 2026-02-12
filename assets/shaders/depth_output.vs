#version 330

// Vertex shader for depth-only rendering
// Outputs world position for depth calculation

in vec3 vertexPosition;

uniform mat4 mvp;

out float fragDepth;

void main() {
    vec4 clipPos = mvp * vec4(vertexPosition, 1.0);
    gl_Position = clipPos;

    // Pass normalized device depth (will be interpolated)
    fragDepth = (clipPos.z / clipPos.w) * 0.5 + 0.5;
}
