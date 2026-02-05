#version 330

in vec3 fragPosition;
in vec2 fragTexCoord;
in vec4 fragColor;
in vec3 fragNormal;

uniform sampler2D texture0;
uniform vec4 colDiffuse;
uniform vec3 viewPos;
uniform vec4 ambient;
uniform vec3 lightDir;
uniform vec4 lightColor;

out vec4 finalColor;

void main()
{
    vec3 normal = normalize(fragNormal);
    vec3 viewDir = normalize(viewPos - fragPosition);
    vec3 lightDirection = normalize(lightDir);

    // Diffuse lighting (wrap lighting for softer look)
    float NdotL = dot(normal, lightDirection);
    float diff = max(NdotL, 0.0);
    // Add slight wrap lighting for softer shadows on geometry
    float wrapDiff = max(NdotL * 0.5 + 0.5, 0.0);
    diff = mix(diff, wrapDiff, 0.3);

    // Specular (Blinn-Phong) - tighter, brighter highlights
    vec3 halfwayDir = normalize(lightDirection + viewDir);
    float NdotH = max(dot(normal, halfwayDir), 0.0);
    float spec = pow(NdotH, 64.0);

    // Fresnel effect - brighter edges
    float fresnel = pow(1.0 - max(dot(normal, viewDir), 0.0), 3.0);

    // Rim lighting
    float rim = fresnel * max(dot(normal, lightDirection), 0.0);

    // Combine lighting components
    vec3 diffuseLight = diff * lightColor.rgb;
    vec3 specularLight = spec * lightColor.rgb * 1.2;
    vec3 rimLight = rim * lightColor.rgb * 0.5;
    vec3 fresnelLight = fresnel * ambient.rgb * 0.3;

    // Final lighting
    vec3 lighting = ambient.rgb + diffuseLight + specularLight + rimLight + fresnelLight;

    // Apply material color
    vec3 baseColor = colDiffuse.rgb * fragColor.rgb;
    vec3 result = lighting * baseColor;

    // Slight tone mapping to prevent over-bright
    result = result / (result + vec3(1.0));

    // Gamma correction
    result = pow(result, vec3(1.0/2.2));

    finalColor = vec4(result, colDiffuse.a);
}
