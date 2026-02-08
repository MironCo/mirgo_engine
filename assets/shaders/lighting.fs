#version 330

in vec3 fragPosition;
in vec2 fragTexCoord;
in vec4 fragColor;
in vec3 fragNormal;
in vec3 fragTangent;
in vec3 fragBitangent;
in vec2 fragShadowTexCoord;
in float fragShadowDepth;

uniform sampler2D texture0;      // Albedo/diffuse texture
uniform sampler2D texture1;      // Normal map (Raylib's MAP_NORMAL slot)
uniform sampler2D shadowMap;
uniform vec4 colDiffuse;
uniform vec3 viewPos;
uniform vec4 ambient;
uniform vec3 lightDir;
uniform vec4 lightColor;

// Material properties
uniform float metallic;   // 0 = diffuse, 1 = metallic
uniform float roughness;  // 0 = shiny, 1 = rough
uniform float emissive;   // emission intensity

out vec4 finalColor;

float calculateShadow(vec3 normal, vec3 lightDirection)
{
    // Check if fragment is outside shadowmap bounds
    if (fragShadowTexCoord.x < 0.0 || fragShadowTexCoord.x > 1.0 ||
        fragShadowTexCoord.y < 0.0 || fragShadowTexCoord.y > 1.0) {
        return 1.0; // No shadow outside the shadowmap
    }

    // Slope-scaled bias: surfaces at steep angles to light need more bias
    float NdotL = dot(normal, lightDirection);
    float bias = max(0.004 * (1.0 - NdotL), 0.0005);

    // PCF: sample a 5x5 kernel around the shadow coordinate
    vec2 texelSize = 1.0 / textureSize(shadowMap, 0);
    float shadow = 0.0;
    for (int x = -2; x <= 2; x++) {
        for (int y = -2; y <= 2; y++) {
            float sampleDepth = texture(shadowMap, fragShadowTexCoord + vec2(x, y) * texelSize).r;
            shadow += (fragShadowDepth - bias > sampleDepth) ? 0.0 : 1.0;
        }
    }
    shadow /= 25.0;

    // Remap from [0,1] to [shadowIntensity,1]
    float shadowIntensity = 0.3;
    return shadowIntensity + shadow * (1.0 - shadowIntensity);
}

void main()
{
    vec3 normal = normalize(fragNormal);

    vec3 viewDir = normalize(viewPos - fragPosition);
    vec3 lightDirection = normalize(-lightDir);

    // Calculate shadow factor (pass normal and light dir for slope-scaled bias)
    float shadow = calculateShadow(normal, lightDirection);

    // Base color - sample albedo texture and multiply by vertex color and diffuse
    vec4 texColor = texture(texture0, fragTexCoord);
    vec3 baseColor = texColor.rgb * colDiffuse.rgb;

    // Diffuse lighting
    float NdotL = dot(normal, lightDirection);
    float diff = max(NdotL, 0.0);

    // Specular (Blinn-Phong) - shininess based on roughness
    vec3 halfwayDir = normalize(lightDirection + viewDir);
    float NdotH = max(dot(normal, halfwayDir), 0.0);
    // Roughness affects specular power: rough = wide soft highlight, smooth = tight sharp highlight
    float shininess = mix(64.0, 4.0, roughness);
    float spec = pow(NdotH, shininess);
    // Roughness also affects specular intensity
    float specIntensity = mix(1.0, 0.1, roughness);

    // Fresnel effect - brighter edges (stronger for metals)
    float fresnel = pow(1.0 - max(dot(normal, viewDir), 0.0), 3.0);
    fresnel = mix(fresnel * 0.5, fresnel, metallic);

    // Rim lighting
    float rim = fresnel * max(dot(normal, lightDirection), 0.0);

    // Metallic affects how specular color is derived
    // Non-metals: white specular, Metals: specular tinted by base color
    vec3 specColor = mix(vec3(1.0), baseColor, metallic);

    // Combine lighting components - apply shadow to diffuse and specular
    vec3 diffuseLight = diff * lightColor.rgb * shadow;
    // Metals have reduced diffuse (energy conservation)
    diffuseLight *= (1.0 - metallic * 0.8);

    vec3 specularLight = spec * specColor * lightColor.rgb * specIntensity * shadow;
    vec3 rimLight = rim * lightColor.rgb * 0.5 * shadow;
    vec3 fresnelLight = fresnel * ambient.rgb * 0.3;

    // Final lighting (ambient is not affected by shadow)
    vec3 lighting = ambient.rgb + diffuseLight + specularLight + rimLight + fresnelLight;

    // Apply material color
    vec3 result = lighting * baseColor;

    // Add emission
    result += baseColor * emissive;

    // Slight tone mapping to prevent over-bright
    result = result / (result + vec3(1.0));

    // Gamma correction
    result = pow(result, vec3(1.0/2.2));

    // DEBUG: Uncomment ONE of these to visualize:
    // result = normal * 0.5 + 0.5;  // Normals as color
    // result = texture(texture1, fragTexCoord).rgb;  // Normal map texture
    // result = fragTangent * 0.5 + 0.5;  // Tangent data

    finalColor = vec4(result, texColor.a * colDiffuse.a);
}
